package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type file struct {
	compressedFile

	fs     *Fs
	isDir  bool
	offset int64
}

var _ afero.File = &file{}

func (f *file) Name() string {
	return f.header.Name
}

func (f *file) Close() error {
	return nil
}

func (f *file) Read(p []byte) (n int, err error) {
	off := f.offset
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

	start := f.start
	r := bytes.NewReader(f.fs.compressedData)
	_, err = r.Seek(start, io.SeekStart)
	if err != nil {
		return 0, err
	}
	compressor, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}
	defer compressor.Close()
	archive := tar.NewReader(compressor)
	header, err := archive.Next()
	if err == io.EOF {
		panic("Known file could not be found after seek")
	}
	if err != nil {
		return 0, err
	}
	if header.Name != f.header.Name {
		return 0, errors.Errorf("Unrecognized file at seek path. Expected %q, found %q.", f.header.Name, header.Name)
	}

	if off == 0 {
		return archive.Read(p)
	}

	buf := make([]byte, f.header.Size)
	_, err = archive.Read(buf)
	if err != nil {
		return 0, err
	}
	return copy(p, buf[off:]), nil
}

func (f *file) Seek(offset int64, whence int) (newOffset int64, err error) {
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = f.offset + offset
	case io.SeekEnd:
		newOffset = f.header.Size + offset
	}
	if newOffset < 0 {
		newOffset = 0
	} else if newOffset >= f.header.Size {
		newOffset = f.header.Size
		err = io.EOF
	}
	f.offset = newOffset
	return
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	names, err := f.Readdirnames(count)
	if err != nil {
		return nil, err
	}
	var infos []os.FileInfo
	for _, name := range names {
		info, err := f.fs.Stat(filepath.Join(f.header.Name, name))
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
	if n > 0 {
		return nil, syscall.ENOSYS
	}

	files := f.fs.directories[f.header.Name]
	var names []string
	for baseName := range files {
		names = append(names, baseName)
	}
	sort.Strings(names)
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
