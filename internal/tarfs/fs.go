package tarfs

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	goPath "path"
	"strings"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const (
	pathSeparatorRune = '/'
	pathSeparator     = string(pathSeparatorRune)
)

type Fs struct {
	// directories holds directory paths and their children's base names. Non-nil means directory exists with no children.
	directories map[string]map[string]bool
	done        <-chan struct{}
	files       map[string]*uncompressedFile
	initErr     error
}

type uncompressedFile struct {
	header   *tar.Header
	contents []byte
}

var _ afero.Fs = &Fs{}

func New(r io.Reader) *Fs {
	done := make(chan struct{})
	fs := &Fs{
		directories: make(map[string]map[string]bool),
		files:       make(map[string]*uncompressedFile),
		done:        done,
	}
	go fs.downloadGzip(r, done)
	return fs
}

func (fs *Fs) downloadGzip(r io.Reader, done chan<- struct{}) {
	err := fs.downloadGzipErr(r)
	if err != nil {
		fs.initErr = err
	}
	close(done)

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
		contents := make([]byte, header.Size)
		_, err = io.ReadFull(archive, contents)
		if err != nil {
			return err
		}

		header.Name = path
		file := &uncompressedFile{
			header:   header,
			contents: contents,
		}

		fs.files[path] = file
		for _, segment := range dirsFromPath(originalName) {
			if fs.directories[segment] == nil {
				fs.directories[segment] = make(map[string]bool, 1)
			}
		}
	}

	// register dirs and files with parent directories
	for dir := range fs.directories {
		parent := goPath.Dir(dir)
		if dir != pathSeparator {
			fs.directories[parent][dir] = true
		}
	}
	for path := range fs.files {
		parent := goPath.Dir(path)
		if path != pathSeparator {
			fs.directories[parent][path] = true
		}
	}
	return nil
}

// dirsFromPath returns all directory segments in 'path'. Assumes 'path' is a raw header name from a tar.
func dirsFromPath(path string) []string {
	var dirs []string
	if strings.HasSuffix(path, pathSeparator) {
		// denotes a tar directory path, so clean it and add it before pop
		path = fsutil.NormalizePath(path)
		dirs = append(dirs, path)
	}
	if path == pathSeparator {
		return dirs
	}
	path = fsutil.NormalizePath(path) // make absolute + clean
	path = goPath.Dir(path)           // pop normal files from the end
	var prevPath string
	for ; path != prevPath; path = goPath.Dir(path) {
		dirs = append(dirs, path)
		prevPath = path
	}
	return dirs
}

func (fs *Fs) ensurePath(path string) error {
	if _, exists := fs.files[path]; exists {
		return fs.initErr
	}
	<-fs.done
	return fs.initErr
}

func (fs *Fs) Open(path string) (afero.File, error) {
	path = fsutil.NormalizePath(path)
	if err := fs.ensurePath(path); err != nil {
		return nil, err
	}
	_, isDir := fs.directories[path]
	if isDir {
		return &file{
			uncompressedFile: &uncompressedFile{
				header: &tar.Header{Name: path}, // just enough to look up more dir info in fs
			},
			fs:    fs,
			isDir: true,
		}, nil
	}
	f, present := fs.files[path]
	if !present {
		return nil, &os.PathError{Op: "open", Path: path, Err: os.ErrNotExist}
	}

	return &file{
		uncompressedFile: f,
		fs:               fs,
		isDir:            f.header.FileInfo().IsDir(),
	}, nil
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag != os.O_RDONLY {
		return nil, syscall.EPERM
	}
	return fs.Open(name)
}

func (fs *Fs) Stat(path string) (os.FileInfo, error) {
	path = fsutil.NormalizePath(path)
	if err := fs.ensurePath(path); err != nil {
		return nil, err
	}
	f, present := fs.files[path]
	if present {
		return f.header.FileInfo(), nil
	}
	_, isDir := fs.directories[path]
	if !isDir {
		return nil, &os.PathError{Op: "stat", Path: path, Err: os.ErrNotExist}
	}

	return &genericDirInfo{name: goPath.Base(path)}, nil
}

func (fs *Fs) Name() string {
	return "tarfs.Fs"
}

func (fs *Fs) Create(name string) (afero.File, error)                      { return nil, syscall.EPERM }
func (fs *Fs) Mkdir(name string, perm os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) MkdirAll(path string, perm os.FileMode) error                { return syscall.EPERM }
func (fs *Fs) Remove(name string) error                                    { return syscall.EPERM }
func (fs *Fs) RemoveAll(path string) error                                 { return syscall.EPERM }
func (fs *Fs) Rename(oldname, newname string) error                        { return syscall.EPERM }
func (fs *Fs) Chmod(name string, mode os.FileMode) error                   { return syscall.EPERM }
func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error { return syscall.EPERM }
