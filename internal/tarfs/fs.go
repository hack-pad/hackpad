package tarfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"path"
	"runtime"
	"sync"

	"github.com/hack-pad/hackpad/internal/bufferpool"
	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/pubsub"
	"github.com/hack-pad/hackpad/log"
	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

type FS struct {
	underlyingFS BaseFS
	ps           pubsub.PubSub
	ctx          context.Context
	cancel       context.CancelFunc
	initErr      error
}

var _ hackpadfs.FS = &FS{}

type BaseFS interface {
	hackpadfs.OpenFileFS
	hackpadfs.ChmodFS
	hackpadfs.MkdirFS
}

func New(r io.Reader, underlyingFS BaseFS) (_ *FS, retErr error) {
	defer func() { retErr = errors.Wrap(retErr, "tarfs") }()

	if dirEntries, err := hackpadfs.ReadDir(underlyingFS, "."); err != nil || len(dirEntries) != 0 {
		var names []string
		for _, dirEntry := range dirEntries {
			names = append(names, dirEntry.Name())
		}
		return nil, errors.Errorf("Root '/' must be an empty directory, got: %T %v %s", underlyingFS, err, names)
	}

	ctx, cancel := context.WithCancel(context.Background())
	fs := &FS{
		underlyingFS: underlyingFS,
		ps:           pubsub.New(ctx),
		ctx:          ctx,
		cancel:       cancel,
	}
	go fs.downloadGzip(r)
	return fs, nil
}

func (fs *FS) downloadGzip(r io.Reader) {
	err := fs.downloadGzipErr(r)
	if err != nil {
		fs.initErr = err
		log.Error("tarfs: Failed to complete overlay: ", err)
	}
	fs.cancel()

	if closer, ok := r.(io.Closer); ok {
		_ = closer.Close()
	}
}

func (fs *FS) downloadGzipErr(r io.Reader) error {
	compressor, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "gzip reader")
	}
	defer compressor.Close()

	archive := tar.NewReader(compressor)
	const (
		mebibyte       = 1 << 20
		kibibyte       = 1 << 10
		maxMemory      = 20 * mebibyte
		bigBufMemory   = 4 * mebibyte
		smallBufMemory = 150 * kibibyte

		// at least a couple big and small buffers, then a large quantity of small ones make up the remainder
		bigBufCount   = 2
		smallBufCount = (maxMemory - bigBufCount*bigBufMemory) / smallBufMemory
	)
	smallPool := bufferpool.New(smallBufMemory, smallBufCount)
	bigPool := bufferpool.New(bigBufMemory, bigBufCount)
	defer runtime.GC() // forcefully clean up memory pools

	mkdirCache := make(map[string]bool)
	cachedMkdirAll := func(path string, perm hackpadfs.FileMode) error {
		if _, ok := mkdirCache[path]; ok {
			return nil
		}
		err := hackpadfs.MkdirAll(fs.underlyingFS, path, perm)
		if err == nil {
			mkdirCache[path] = true
		}
		return err
	}

	var wg sync.WaitGroup
	errs := make(chan error, 1)
	for {
		select {
		case err := <-errs:
			return err
		default:
		}
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "next tar file")
		}
		err = fs.initProcessFile(header, archive, &wg, errs, cachedMkdirAll, bigPool, smallPool)
		if err != nil {
			return err
		}
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case err := <-errs:
		return err
	case <-done:
		return nil
	}
}

func (fs *FS) initProcessFile(
	header *tar.Header, r io.Reader,
	wg *sync.WaitGroup, errs chan error,
	mkdirAll func(string, hackpadfs.FileMode) error,
	bigPool, smallPool *bufferpool.Pool,
) error {
	select {
	case <-fs.ctx.Done():
		return fs.ctx.Err()
	default:
	}

	originalName := header.Name
	p := common.ResolvePath(".", originalName)
	info := header.FileInfo()

	dir := path.Dir(p)
	err := mkdirAll(dir, 0700)
	if err != nil {
		return errors.Wrap(err, "prepping base dir")
	}

	if info.IsDir() {
		// assume dir does not exist yet, then chmod if it does exist
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := fs.underlyingFS.Mkdir(p, info.Mode())
			if err != nil {
				if !errors.Is(err, hackpadfs.ErrExist) {
					errs <- errors.Wrap(err, "copying dir")
					return
				}
				err = fs.underlyingFS.Chmod(p, info.Mode())
				if err != nil {
					errs <- errors.Wrap(err, "copying dir")
					return
				}
			}
		}()
		return nil
	}

	reader := fullReader{r} // fullReader: call f.Write as few times as possible, since large files are expensive
	// read once. if we reached EOF, then write it to fs asynchronously
	smallBuf := smallPool.Wait()
	n, err := reader.Read(smallBuf.Data)
	switch err {
	case io.EOF:
		wg.Add(1)
		go func() {
			err := fs.writeFile(p, info, smallBuf, n, nil, nil)
			smallBuf.Done()
			if err != nil {
				errs <- err
			}
			wg.Done()
		}()
		return nil
	case nil:
		bigBuf := bigPool.Wait()
		err := fs.writeFile(p, info, smallBuf, n, reader, bigBuf)
		bigBuf.Done()
		smallBuf.Done()
		return err
	default:
		return err
	}
}

func (fs *FS) writeFile(path string, info hackpadfs.FileInfo, initialBuf *bufferpool.Buffer, n int, r io.Reader, copyBuf *bufferpool.Buffer) (returnedErr error) {
	f, err := fs.underlyingFS.OpenFile(path, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate|hackpadfs.FlagTruncate, info.Mode())
	if err != nil {
		return errors.Wrap(err, "opening destination file")
	}
	defer func() {
		f.Close()
		if returnedErr == nil {
			fs.ps.Emit(path) // only emit for non-dirs, dirs will wait until the download completes to ensure correctness
		}
	}()

	fWriter, ok := f.(io.Writer)
	if !ok {
		return hackpadfs.ErrNotImplemented
	}

	_, err = fWriter.Write(initialBuf.Data[:n])
	if err != nil {
		return errors.Wrap(err, "write: copying file")
	}

	if r == nil {
		// a nil reader signals we already did a read of N bytes and hit EOF,
		// so the above copy is sufficient, return now
		return nil
	}

	_, err = io.CopyBuffer(fWriter, r, copyBuf.Data)
	return errors.Wrap(err, "copybuf: copying file")
}

type fullReader struct {
	io.Reader
}

func (f fullReader) Read(p []byte) (n int, err error) {
	n, err = io.ReadFull(f.Reader, p)
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return
}

func (fs *FS) Open(name string) (hackpadfs.File, error) {
	if !hackpadfs.ValidPath(name) {
		return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: hackpadfs.ErrInvalid}
	}
	fs.ps.Wait(name)
	if fs.initErr != nil {
		return nil, &hackpadfs.PathError{Op: "open", Path: name, Err: fs.initErr}
	}
	return fs.underlyingFS.Open(name)
}

func (fs *FS) Done() <-chan struct{} {
	return fs.ctx.Done()
}

func (fs *FS) InitErr() error {
	return fs.initErr
}

type clearFS interface {
	Clear(ctx context.Context) error
}

func (fs *FS) Clear(ctx context.Context) (err error) {
	if clearFS, ok := fs.underlyingFS.(clearFS); ok {
		fs.initErr = context.Canceled
		fs.cancel()
		return clearFS.Clear(ctx)
	}
	return errors.New("Unsupported operation for base FS")
}
