package fs

import (
	"io"
	"os"

	"github.com/johnstarich/go-wasm/internal/interop"
	"go.uber.org/atomic"
)

var lastPipeNumber = atomic.NewUint64(0)

func (f *FileDescriptors) Pipe() [2]FID {
	r, w := newPipe(f.newFID)
	f.addFileDescriptor(r)
	f.addFileDescriptor(w)
	r.Open(f.parentPID)
	w.Open(f.parentPID)
	return [2]FID{r.id, w.id}
}

func newPipe(newFID func() FID) (r, w *fileDescriptor) {
	pipeC := newPipeChan()
	readerFID, writerFID := newFID(), newFID()
	r = newIrregularFileDescriptor(readerFID, &pipeReadOnly{pipeChan: pipeC, fid: readerFID}, os.ModeNamedPipe)
	w = newIrregularFileDescriptor(writerFID, &pipeWriteOnly{pipeChan: pipeC, fid: writerFID}, os.ModeNamedPipe)
	return
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
		name: "pipe",
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

	fid FID
}

func (r *pipeReadOnly) Name() string {
	return "pipe" + r.fid.String()
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

	fid FID
}

func (w *pipeWriteOnly) Name() string {
	return "pipe" + w.fid.String()
}

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}
