package fs

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpad/internal/interop"
)

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
	rPipe := &namedPipe{pipeChan: pipeC, fid: readerFID}
	r = newIrregularFileDescriptor(
		readerFID,
		rPipe.Name(),
		&pipeReadOnly{rPipe},
		os.ModeNamedPipe,
	)
	wPipe := &namedPipe{pipeChan: pipeC, fid: writerFID}
	w = newIrregularFileDescriptor(
		writerFID,
		wPipe.Name(),
		&pipeWriteOnly{wPipe},
		os.ModeNamedPipe,
	)
	return
}

type pipeChan struct {
	buf            chan byte
	done           chan struct{}
	reader, writer FID
}

func newPipeChan(reader, writer FID) *pipeChan {
	const maxPipeBuffer = 32 << 10 // 32KiB
	return &pipeChan{
		buf:    make(chan byte, maxPipeBuffer),
		done:   make(chan struct{}),
		reader: reader,
		writer: writer,
	}
}

type pipeStat struct {
	name string
	size int64
	mode os.FileMode
}

func (p pipeStat) Name() string       { return p.name }
func (p pipeStat) Size() int64        { return p.size }
func (p pipeStat) Mode() os.FileMode  { return p.mode }
func (p pipeStat) ModTime() time.Time { return time.Time{} }
func (p pipeStat) IsDir() bool        { return false }
func (p pipeStat) Sys() interface{}   { return nil }

func (p *pipeChan) Stat() (os.FileInfo, error) {
	return &pipeStat{
		name: "",
		size: int64(len(p.buf)),
		mode: os.ModeNamedPipe,
	}, nil
}

func (p *pipeChan) Sync() error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		return nil
	case <-timer.C:
		return io.ErrNoProgress
	}
}

func (p *pipeChan) Read(buf []byte) (n int, err error) {
	for n < len(buf) {
		// Read should always block if the pipe is not closed
		b, ok := <-p.buf
		if !ok {
			err = io.EOF
			return
		}
		buf[n] = b
		n++
	}
	if n == 0 {
		err = io.EOF
	}
	return
}

func (p *pipeChan) Write(buf []byte) (n int, err error) {
	for _, b := range buf {
		select {
		case <-p.done:
			// do not allow writes to a closed pipe
			return 0, interop.BadFileNumber(p.writer)
		case p.buf <- b:
			n++
			// no default case allowed, Write should always return immediately if the pipe buffer has space, otherwise it should block
		}
	}
	if n < len(buf) {
		err = io.ErrShortWrite
	}
	return
}

func (p *pipeChan) Close() error {
	select {
	case <-p.done:
		return interop.BadFileNumber(p.writer)
	default:
		close(p.done)
		close(p.buf)
		return nil
	}
}

type namedPipe struct {
	*pipeChan
	fid FID
}

func (n *namedPipe) Name() string {
	return "pipe" + n.fid.String()
}

type pipeReadOnly struct {
	*namedPipe
}

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

type pipeWriteOnly struct {
	*namedPipe
}

func (w *pipeWriteOnly) Read(buf []byte) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (w *pipeWriteOnly) WriteAt(buf []byte, off int64) (n int, err error) {
	if off == 0 {
		return w.Write(buf)
	}
	return 0, interop.ErrNotImplemented
}
