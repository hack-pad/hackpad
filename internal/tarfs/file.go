package tarfs

import (
	"io"
	"os"
	"path"
	"sort"
	"sync/atomic"
	"syscall"

	"github.com/spf13/afero"
)

type file struct {
	*uncompressedFile

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
	if off >= int64(len(f.contents)) {
		return 0, io.EOF
	}
	return copy(p, f.contents[off:]), err
}

func (f *file) Seek(offset int64, whence int) (newOffset int64, err error) {
	offsetPtr := &f.offset
	size := f.header.Size
	if f.isDir {
		offsetPtr = &f.dirOffset
		size = int64(len(f.dirFiles()))
	}
	curOffset := atomic.LoadInt64(offsetPtr)
	switch whence {
	case io.SeekStart:
		newOffset = curOffset
	case io.SeekCurrent:
		newOffset = curOffset + offset
	case io.SeekEnd:
		newOffset = size + offset
	}
	if newOffset < 0 {
		newOffset = 0
	} else if newOffset >= size {
		newOffset = size
		err = io.EOF
	}
	atomic.CompareAndSwapInt64(offsetPtr, curOffset, newOffset)
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

	files := f.dirFiles()
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

// dirFiles returns the files in this directory (check f.isDir first)
func (f *file) dirFiles() map[string]bool {
	return f.fs.directories[f.header.Name]
}

func (f *file) Stat() (os.FileInfo, error) {
	return f.fs.Stat(f.header.Name)
}

func (f *file) Write(p []byte) (n int, err error)              { return 0, syscall.EPERM }
func (f *file) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EPERM }
func (f *file) Sync() error                                    { return syscall.EPERM }
func (f *file) Truncate(size int64) error                      { return syscall.EPERM }
func (f *file) WriteString(s string) (ret int, err error)      { return 0, syscall.EPERM }
