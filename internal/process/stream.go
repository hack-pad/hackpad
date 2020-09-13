package process

import (
	"context"
	"io"
	"sync"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
)

const (
	eventError = "error"
	eventData  = "data"
	eventEnd   = "end"
	eventClose = "close"
)

type readableStream struct {
	interop.EventTarget
	target    js.Value
	startData sync.Once
	reader    io.Reader
	onJSFunc  js.Func
}

func newReadableStream(ctx context.Context, r io.Reader, target js.Value) *readableStream {
	stream := &readableStream{
		EventTarget: interop.NewEventTarget(),
		target:      target,
		reader:      r,
	}
	stream.onJSFunc = js.FuncOf(stream.jsOn)
	go func() {
		<-ctx.Done()
		stream.onJSFunc.Release()
	}()
	return stream
}

func (r *readableStream) jsOn(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return interop.WrapAsJSError(interop.ErrNotImplemented, "Not enough args. Pass an event name and a callback.")
	}
	eventName := args[0].String()
	callback := args[1]
	r.Listen(eventName, func(event interop.Event, args ...interface{}) {
		callback.Invoke(args...)
	})
	if eventName == eventData {
		r.startData.Do(func() {
			go r.runData()
		})
	}
	return nil
}

func (r *readableStream) runData() {
	err := r.runDataErr()
	if err == io.EOF {
		r.Emit(interop.Event{
			Target: r.target,
			Type:   "end",
		})
	} else if err != nil {
		r.Emit(interop.Event{
			Target: r.target,
			Type:   "error",
		}, err)
	}
	r.Emit(interop.Event{
		Target: r.target,
		Type:   "close",
	})
}

func (r *readableStream) runDataErr() error {
	const maxBufSize = 1
	buf := make([]byte, maxBufSize)
	for {
		log.Printf("Reading %d bytes", maxBufSize)
		n, err := r.reader.Read(buf)
		if err != nil {
			return err
		}
		log.Print("Read data:", buf)
		r.Emit(interop.Event{
			Target: r.target,
			Type:   "data",
		}, buf[:n])
	}
}

func (r *readableStream) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"on": r.onJSFunc,
	})
}

type writableStream struct {
	io.Writer
	writeJSFunc js.Func
}

func newWritableStream(ctx context.Context, w io.Writer) *writableStream {
	stream := &writableStream{
		Writer: w,
	}
	stream.writeJSFunc = js.FuncOf(stream.jsWrite)
	go func() {
		<-ctx.Done()
		stream.writeJSFunc.Release()
	}()
	return stream
}

func (w *writableStream) jsWrite(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return interop.WrapAsJSError(interop.ErrNotImplemented, "Not enough args. Pass a chunk to write to the stream.")
	}
	chunk := args[0]
	var buf []byte
	if chunk.Type() == js.TypeString {
		buf = []byte(chunk.String())
	} else {
		buf = make([]byte, chunk.Length())
		js.CopyBytesToGo(buf, chunk)
	}
	log.Print("Writing data:", buf)
	_, err := w.Write(buf)
	return interop.WrapAsJSError(err, "Failed to write to stream")
}

func (w *writableStream) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"write": w.writeJSFunc,
	})
}
