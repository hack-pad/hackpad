package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path"
	"sort"
	"sync/atomic"
	"syscall"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type file struct {
	compressedFile

	fs        *Fs
	isDir     bool
	offset    int64
	dirOffset int64
}

var _ afero.File = &file{}

func (f *file) Name() string {
	return f.header.Name
}

func (f *file) Close() error {
	return nil
}

func (f *file) Read(p []byte) (n int, err error) {
	off := atomic.LoadInt64(&f.offset)
	n, err = f.ReadAt(p, off)
	atomic.CompareAndSwapInt64(&f.offset, off, off+int64(n))
	return
}

func (f *file) ReadAt(p []byte, off int64) (n int, err error) {
	if f.isDir {
		return 0, syscall.EISDIR
	}
	if off >= f.header.Size {
		return 0, io.EOF
	}

	r := bytes.NewReader(f.fs.compressedData)
	compressor, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer compressor.Close()
	archive := tar.NewReader(compressor)
	var header *tar.Header
	for {
		header, err = archive.Next()
		if err == io.EOF {
			panic("Known file could not be found")
		}
		if err != nil {
			return 0, err
		}
		if header.Name == f.header.Name {
			break
		}
	}
	if header == nil {
		panic("Known file could not be found")
	}
	if header.Name != f.header.Name {
		return 0, errors.Errorf("Unrecognized file at seek path. Expected %q, found %q.", f.header.Name, header.Name)
	}

	if off == 0 {
		return archive.Read(p)
	}

	buf := make([]byte, f.header.Size)
	_, err = archive.Read(buf)
	return copy(p, buf[off:]), err
}

func (f *file) Seek(offset int64, whence int) (newOffset int64, err error) {
	curOffset := atomic.LoadInt64(&f.offset)
	switch whence {
	case io.SeekStart:
		newOffset = curOffset
	case io.SeekCurrent:
		newOffset = curOffset + offset
	case io.SeekEnd:
		newOffset = f.header.Size + offset
	}
	if newOffset < 0 {
		newOffset = 0
	} else if newOffset >= f.header.Size {
		newOffset = f.header.Size
		err = io.EOF
	}
	atomic.CompareAndSwapInt64(&f.offset, curOffset, newOffset)
	return
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	names, err := f.Readdirnames(count)
	if err != nil {
		return nil, err
	}
	var infos []os.FileInfo
	for _, name := range names {
		info, err := f.fs.Stat(path.Join(f.header.Name, name))
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (f *file) Readdirnames(n int) ([]string, error) {
	if !f.isDir {
		return nil, syscall.ENOTDIR
	}

	files := f.fs.directories[fsutil.NormalizePath(f.header.Name)]
	var names []string
	for name := range files {
		names = append(names, path.Base(name))
	}
	sort.Strings(names)

	off := atomic.LoadInt64(&f.dirOffset)
	remaining := int64(len(names)) - off
	switch {
	case n <= 0 && remaining == 0:
		return []string{}, nil
	case remaining == 0:
		return []string{}, io.EOF
	case n <= 0:
		atomic.StoreInt64(&f.dirOffset, int64(len(names)))
		return names, nil
	}

	bigN := int64(n)
	if bigN > remaining {
		bigN = remaining
	}
	names = names[off : off+bigN]
	atomic.CompareAndSwapInt64(&f.dirOffset, off, off+bigN)
	return names, nil
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.fs.Stat(f.header.Name)
}

func (f *file) Write(p []byte) (n int, err error)              { return 0, syscall.EPERM }
func (f *file) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EPERM }
func (f *file) Sync() error                                    { return syscall.EPERM }
func (f *file) Truncate(size int64) error                      { return syscall.EPERM }
func (f *file) WriteString(s string) (ret int, err error)      { return 0, syscall.EPERM }
