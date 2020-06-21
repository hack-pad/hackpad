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
	readerFID, writerFID := newFID(), newFID()
	pipeC := newPipeChan(readerFID, writerFID)
	r = newIrregularFileDescriptor(readerFID, &pipeReadOnly{pipeChan: pipeC, fid: readerFID}, os.ModeNamedPipe)
	w = newIrregularFileDescriptor(writerFID, &pipeWriteOnly{pipeChan: pipeC, fid: writerFID}, os.ModeNamedPipe)
	return
}

type pipeChan struct {
	unimplementedFile

	buf            chan byte
	closed         bool
	reader, writer FID
}

func newPipeChan(reader, writer FID) *pipeChan {
	const maxPipeBuffer = 32 << 10 // 32KiB
	return &pipeChan{
		buf:    make(chan byte, maxPipeBuffer),
		reader: reader,
		writer: writer,
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
			return 0, interop.BadFileNumber(p.writer)
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
		return interop.BadFileNumber(p.writer)
	}
	p.closed = true
	close(p.buf)
	return nil
}

type namedPipe struct {
	*pipeChan
	fid FID
}

func (n *namedPipe) Name() string {
	return "pipe" + n.fid.String()
}

type pipeReadOnly namedPipe

func (r *pipeReadOnly) ReadAt(buf []byte, off int64) (n int, err error) {
	if off == 0 {
		return r.Read(buf)
	}
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Write(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (r *pipeReadOnly) Close() error {
	// only write side of pipe should close the buffer
	return nil
}

type pipeWriteOnly namedPipe

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (w *pipeWriteOnly) WriteAt(buf []byte, off int64) (n int, err error) {
	if off == 0 {
		return w.Write(buf)
	}
	return 0, interop.ErrNotImplemented
}
