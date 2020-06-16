package fs

import (
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func pipe(args []js.Value) ([]interface{}, error) {
	fds, err := pipeSync(args)
	return []interface{}{fds}, err
}

func pipeSync(args []js.Value) (interface{}, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("Invalid number of args, expected 0: %v", args)
	}
	fds, err := Pipe()
	if err != nil {
		return nil, err
	}
	return []interface{}{fds[0], fds[1]}, err
}

func Pipe() ([]uint64, error) {
	f := newPipeChan()
	r, err := openWithFile(readOnly{f})
	if err != nil {
		return nil, err
	}
	w, err := openWithFile(writeOnly{f})
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
