package fs

import (
	"io"
	"os"
	"strconv"

	"github.com/johnstarich/go-wasm/internal/interop"
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

	name          string
	buf           chan byte
	processOpened map[uint64]bool
	processClosed map[uint64]bool
	closed        bool
}

func newPipeChan() *pipeChan {
	const maxPipeBuffer = 4096
	return &pipeChan{
		name:          "pipe" + strconv.FormatUint(lastPipeNumber.Inc(), 10),
		buf:           make(chan byte, maxPipeBuffer),
		processOpened: make(map[uint64]bool),
		processClosed: make(map[uint64]bool),
	}
}

func (p *pipeChan) OpenPID(pid uint64) {
	p.processOpened[pid] = true
}

func (p *pipeChan) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		select {
		case b := <-p.buf:
			buf[n] = b
			n++
		default:
			if p.closed {
				err = io.EOF
			}
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
	if len(p.processClosed) != len(p.processOpened) {
		return nil
	}
	p.closed = true
	close(p.buf)
	return nil
}

func (p *pipeChan) Name() string {
	return p.name
}

type pipeReadOnly struct {
	*pipeChan
}

func (r *pipeReadOnly) Write(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Close() error {
	// only write side of pipe should close the buffer
	return nil
}

func (r *pipeReadOnly) ClosePID(pid uint64) error {
	return r.pipeChan.Close()
}

type pipeWriteOnly struct {
	*pipeChan
}

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (w *pipeWriteOnly) ClosePID(pid uint64) error {
	w.pipeChan.processClosed[pid] = true
	return w.pipeChan.Close()
}
