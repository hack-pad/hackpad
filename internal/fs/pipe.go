package fs

import (
	"os"
	"strconv"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

var lastPipeNumber = atomic.NewUint64(0)

func (f *FileDescriptors) Pipe() ([]FID, error) {
	pipeC := newPipeChan()
	r, err := f.openWithFile(&pipeReadOnly{pipeC}, true)
	if err != nil {
		return nil, err
	}
	w, err := f.openWithFile(&pipeWriteOnly{pipeC}, true)
	return []FID{r, w}, err
}

type pipeChan struct {
	unimplementedFile

	name   string
	buf    chan byte
	closed bool
}

func newPipeChan() afero.File {
	const maxPipeBuffer = 4096
	return &pipeChan{
		name: "pipe" + strconv.FormatUint(lastPipeNumber.Inc(), 10),
		buf:  make(chan byte, maxPipeBuffer),
	}
}

func (p *pipeChan) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		select {
		case b := <-p.buf:
			buf[n] = b
			n++
		default:
			return
		}
	}
	return
}

func (p *pipeChan) Write(buf []byte) (n int, err error) {
	for _, b := range buf {
		if p.closed {
			return 0, os.ErrClosed
		}
		select {
		case p.buf <- b:
			n++
		default:
			return
		}
	}
	return
}

func (p *pipeChan) Close() error {
	p.closed = true
	close(p.buf)
	return nil
}

func (p *pipeChan) Name() string {
	return p.name
}

type pipeReadOnly struct {
	afero.File
}

func (r *pipeReadOnly) Write(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Close() error {
	// only write side of pipe should close the buffer
	return nil
}

type pipeWriteOnly struct {
	afero.File
}

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}
