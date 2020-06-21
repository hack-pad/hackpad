package fs

import (
	"io"
	"os"
	"strconv"

	"github.com/johnstarich/go-wasm/internal/interop"
	"go.uber.org/atomic"
)

var lastPipeNumber = atomic.NewUint64(0)

func (f *FileDescriptors) Pipe() [2]FID {
	pipeC := newPipeChan()
	r := newIrregularFileDescriptor(f.newFID(), &pipeReadOnly{pipeC}, os.ModeNamedPipe)
	f.addFileDescriptor(r)
	w := newIrregularFileDescriptor(f.newFID(), &pipeWriteOnly{pipeC}, os.ModeNamedPipe)
	f.addFileDescriptor(w)
	return [2]FID{r.id, w.id}
}

type pipeChan struct {
	unimplementedFile

	name   string
	buf    chan byte
	closed bool
}

func newPipeChan() *pipeChan {
	const maxPipeBuffer = 32 << 10 // 32KiB
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
			return 0, interop.BadFileErr(p.name)
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
	if p.closed {
		return interop.BadFileErr(p.name)
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
