package fs

import (
	"os"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
)

func (f *FileDescriptors) Pipe() ([]uint64, error) {
	pipeC := newPipeChan()
	r, err := f.openWithFile(readOnly{pipeC})
	if err != nil {
		return nil, err
	}
	w, err := f.openWithFile(writeOnly{pipeC})
	return []uint64{r, w}, err
}

type pipeChan chan byte

func newPipeChan() afero.File {
	const maxPipeBuffer = 4096
	return pipeChan(make(chan byte, maxPipeBuffer))
}

func (p pipeChan) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		select {
		case b := <-p:
			buf[n] = b
			n++
		default:
			return
		}
	}
	return
}

func (p pipeChan) Write(buf []byte) (n int, err error) {
	for _, b := range buf {
		select {
		case p <- b:
			n++
		default:
			return
		}
	}
	return
}

func (p pipeChan) Close() error {
	close(p)
	return nil
}

func (p pipeChan) Name() string {
	return "pipe"
}

func (p pipeChan) ReadAt(b []byte, off int64) (n int, err error) { return 0, interop.ErrNotImplemented }
func (p pipeChan) WriteAt(b []byte, off int64) (n int, err error) {
	return 0, interop.ErrNotImplemented
}
func (p pipeChan) Seek(offset int64, whence int) (int64, error) { return 0, interop.ErrNotImplemented }
func (p pipeChan) Readdir(count int) ([]os.FileInfo, error)     { return nil, interop.ErrNotImplemented }
func (p pipeChan) Readdirnames(n int) ([]string, error)         { return nil, interop.ErrNotImplemented }
func (p pipeChan) Stat() (os.FileInfo, error)                   { return nil, interop.ErrNotImplemented }
func (p pipeChan) Sync() error                                  { return interop.ErrNotImplemented }
func (p pipeChan) Truncate(size int64) error                    { return interop.ErrNotImplemented }
func (p pipeChan) WriteString(s string) (ret int, err error)    { return 0, interop.ErrNotImplemented }

type readOnly struct {
	afero.File
}

func (r readOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

type writeOnly struct {
	afero.File
}

func (w writeOnly) Write(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}
