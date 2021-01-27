package tarfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/bufferpool"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/pubsub"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	pathSeparatorRune = '/'
	pathSeparator     = string(pathSeparatorRune)
)

type Fs struct {
	underlyingFs, underlyingReadOnlyFs afero.Fs
	ps                                 pubsub.PubSub
	done                               context.CancelFunc
	initErr                            error
}

var _ afero.Fs = &Fs{}

func New(r io.Reader, underlyingFs afero.Fs) (_ *Fs, retErr error) {
	defer func() { retErr = errors.Wrap(retErr, "tarfs") }()

	err := underlyingFs.MkdirAll("/", 0700) // ensure root exists
	if err != nil {
		return nil, errors.Wrap(err, "Failed to ensure root '/' directory on underlying FS")
	}

	root, err := underlyingFs.Open("/")
	if err != nil {
		return nil, errors.Wrap(err, "Error reading root '/' on underlying FS")
	}
	defer root.Close()
	if names, err := root.Readdirnames(-1); err != nil || len(names) != 0 {
		return nil, errors.New("Root '/' must be an empty directory")
	}

	ctx, cancel := context.WithCancel(context.Background())
	fs := &Fs{
		underlyingFs:         underlyingFs,
		underlyingReadOnlyFs: afero.NewReadOnlyFs(underlyingFs),
		ps:                   pubsub.New(ctx),
		done:                 cancel,
	}
	go fs.downloadGzip(r)
	return fs, nil
}

func (fs *Fs) downloadGzip(r io.Reader) {
	err := fs.downloadGzipErr(r)
	if err != nil {
		fs.initErr = err
		log.Error("tarfs: Failed to complete overlay: ", err)
	}
	fs.done()

	if closer, ok := r.(io.Closer); ok {
		_ = closer.Close()
	}
}

func (fs *Fs) downloadGzipErr(r io.Reader) error {
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
		bigBufCount   = maxMemory/bigBufMemory - 2
		smallBufCount = (maxMemory - bigBufCount*bigBufMemory) / smallBufMemory
	)
	smallPool := bufferpool.New(smallBufMemory, smallBufCount)
	bigPool := bufferpool.New(bigBufMemory, bigBufCount)
	defer runtime.GC() // forcefully clean up memory pools

	mkdirCache := make(map[string]bool)
	cachedMkdirAll := func(path string, perm os.FileMode) error {
		if _, ok := mkdirCache[path]; ok {
			return nil
		}
		err := fs.underlyingFs.MkdirAll(path, perm)
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

		originalName := header.Name
		path := fsutil.NormalizePath(originalName)
		info := header.FileInfo()

		dir := fsutil.NormalizePath(filepath.Dir(path))
		err = cachedMkdirAll(dir, 0700)
		if err != nil {
			return errors.Wrap(err, "prepping base dir")
		}

		if info.IsDir() {
			// assume dir does not exist yet, then chmod if it does exist
			err = fs.underlyingFs.Mkdir(path, info.Mode())
			if err != nil {
				if !os.IsExist(err) {
					return errors.Wrap(err, "copying dir")
				}
				err = fs.underlyingFs.Chmod(path, info.Mode())
				if err != nil {
					return errors.Wrap(err, "copying dir")
				}
			}
		} else {
			reader := fullReader{archive} // fullReader: call f.Write as few times as possible, since large files are expensive
			// read once. if we reached EOF, then write it to fs asynchronously
			smallBuf := smallPool.Wait()
			n, err := reader.Read(smallBuf.Data)
			switch err {
			case io.EOF:
				wg.Add(1)
				go func() {
					err := fs.writeFile(path, info, smallBuf, n, nil, nil)
					smallBuf.Done()
					if err != nil {
						errs <- err
					}
					wg.Done()
				}()
			case nil:
				bigBuf := bigPool.Wait()
				err := fs.writeFile(path, info, smallBuf, n, reader, bigBuf)
				bigBuf.Done()
				smallBuf.Done()
				if err != nil {
					return err
				}
			default:
				return err
			}
		}
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	for {
		select {
		case err := <-errs:
			return err
		case <-done:
			return nil
		}
	}
}

func (fs *Fs) writeFile(path string, info os.FileInfo, initialBuf *bufferpool.Buffer, n int, r io.Reader, copyBuf *bufferpool.Buffer) (returnedErr error) {
	f, err := fs.underlyingFs.OpenFile(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, info.Mode())
	if err != nil {
		return errors.Wrap(err, "opening destination file")
	}
	defer func() {
		f.Close()
		if returnedErr == nil {
			fs.ps.Emit(path) // only emit for non-dirs, dirs will wait until the download completes to ensure correctness
		}
	}()

	_, err = f.Write(initialBuf.Data[:n])
	if err != nil {
		return errors.Wrap(err, "write: copying file")
	}

	if r == nil {
		// a nil reader signals we already did a read of N bytes and hit EOF,
		// so the above copy is sufficient, return now
		return nil
	}

	_, err = io.CopyBuffer(f, r, copyBuf.Data)
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

func (fs *Fs) ensurePath(path string) (normalizedPath string, err error) {
	path = fsutil.NormalizePath(path)
	fs.ps.Wait(path)
	return path, fs.initErr
}

func (fs *Fs) Open(path string) (afero.File, error) {
	path, err := fs.ensurePath(path)
	if err != nil {
		return nil, err
	}
	return fs.underlyingReadOnlyFs.Open(path)
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	name, err := fs.ensurePath(name)
	if err != nil {
		return nil, err
	}
	return fs.underlyingReadOnlyFs.OpenFile(name, flag, perm)
}

func (fs *Fs) Stat(path string) (os.FileInfo, error) {
	path, err := fs.ensurePath(path)
	if err != nil {
		return nil, err
	}
	return fs.underlyingReadOnlyFs.Stat(path)
}

func (fs *Fs) Name() string {
	return fmt.Sprintf("tarfs.Fs(%q)", fs.underlyingFs.Name())
}

func (fs *Fs) Create(name string) (afero.File, error)                      { return nil, syscall.EPERM }
func (fs *Fs) Mkdir(name string, perm os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) MkdirAll(path string, perm os.FileMode) error                { return syscall.EPERM }
func (fs *Fs) Remove(name string) error                                    { return syscall.EPERM }
func (fs *Fs) RemoveAll(path string) error                                 { return syscall.EPERM }
func (fs *Fs) Rename(oldname, newname string) error                        { return syscall.EPERM }
func (fs *Fs) Chmod(name string, mode os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error { return syscall.EPERM }
