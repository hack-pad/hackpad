package tarfs

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

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

func New(r io.Reader, undleryingFs afero.Fs) (_ *Fs, retErr error) {
	defer func() { retErr = errors.Wrap(retErr, "tarfs") }()

	root, err := undleryingFs.Open("/")
	if err != nil {
		return nil, errors.Wrap(err, "Error reading root '/' on underlying FS")
	}
	defer root.Close()
	if names, err := root.Readdirnames(-1); err != nil || len(names) != 0 {
		return nil, errors.New("Root '/' must be an empty directory")
	}

	ctx, cancel := context.WithCancel(context.Background())
	fs := &Fs{
		underlyingFs:         undleryingFs,
		underlyingReadOnlyFs: afero.NewReadOnlyFs(undleryingFs),
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
	for {
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
		err = fs.underlyingFs.MkdirAll(dir, 0700)
		if err != nil {
			return errors.Wrap(err, "prepping base dir")
		}

		if info.IsDir() {
			destInfo, err := fs.underlyingFs.Stat(path)
			if err == nil && destInfo.IsDir() {
				err = fs.underlyingFs.Chmod(path, info.Mode())
			} else {
				err = fs.underlyingFs.Mkdir(path, info.Mode())
			}
			if err != nil {
				return errors.Wrap(err, "copying dir")
			}
		} else {
			f, err := fs.underlyingFs.OpenFile(path, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_TRUNC, info.Mode())
			if err != nil {
				return errors.Wrap(err, "opening destination file")
			}
			_, err = io.Copy(f, archive)
			if err != nil {
				f.Close()
				return errors.Wrap(err, "copying file")
			}
			f.Close()
			fs.ps.Emit(path) // only emit for non-dirs, dirs will wait until the download completes to ensure correctness
		}
	}
	return nil
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
