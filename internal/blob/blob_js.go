// +build js

package blob

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var uint8Array = js.Global().Get("Uint8Array")

type jsExtensions js.Wrapper

type blob struct {
	bytes    atomic.Value
	hasBytes atomic.Bool
	jsValue  atomic.Value
	hasJS    atomic.Bool
	length   atomic.Int64
}

func NewFromBytes(buf []byte) Blob {
	b := &blob{}
	b.hasBytes.Store(true)
	b.bytes.Store(buf)
	b.length.Store(int64(len(buf)))
	return b
}

func NewFromJS(buf js.Value) (Blob, error) {
	if !buf.Truthy() {
		return NewFromBytes(nil), nil
	}
	if !buf.InstanceOf(uint8Array) {
		return nil, errors.Errorf("Invalid JS array type: %v", buf)
	}
	b := &blob{}
	b.hasJS.Store(true)
	b.jsValue.Store(buf)
	b.length.Store(int64(buf.Length()))
	return b, nil
}

func NewJSLength(length int) Blob {
	buf, err := NewFromJS(uint8Array.New(length))
	if err != nil {
		panic("blob: New empty array of correct type was rejected: " + err.Error())
	}
	return buf
}

func (b *blob) Bytes() []byte {
	if b.hasBytes.Load() {
		return b.bytes.Load().([]byte)
	}
	jsBuf := b.jsValue.Load().(js.Value)
	buf := make([]byte, jsBuf.Length())
	js.CopyBytesToGo(buf, jsBuf)
	b.bytes.Store(buf)
	b.hasBytes.Store(true)
	return buf
}

func (b *blob) JSValue() js.Value {
	if b.hasJS.Load() {
		return b.jsValue.Load().(js.Value)
	}
	buf := b.bytes.Load().([]byte)
	jsBuf := uint8Array.New(len(buf))
	js.CopyBytesToJS(jsBuf, buf)

	b.jsValue.Store(jsBuf)
	b.hasJS.Store(true)
	return jsBuf
}

func (b *blob) Len() int {
	return int(b.length.Load())
}

func (b *blob) View(start, end int64) (_ Blob, returnedErr error) {
	defer common.CatchException(&returnedErr)

	if start == 0 && end == b.length.Load() {
		return b, nil
	}
	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		return NewFromBytes(buf[start:end]), nil
	}
	buf := b.jsValue.Load().(js.Value)
	return NewFromJS(buf.Call("subarray", start, end))
}

func (b *blob) Slice(start, end int64) (_ Blob, returnedErr error) {
	defer common.CatchException(&returnedErr)

	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		bufCopy := make([]byte, end-start)
		copy(bufCopy, buf)
		return NewFromBytes(bufCopy), nil
	}
	buf := b.jsValue.Load().(js.Value)
	return NewFromJS(buf.Call("slice", start, end))
}

func (b *blob) Set(w Blob, off int64) (n int, returnedErr error) {
	defer common.CatchException(&returnedErr)

	// TODO need better consistency if this is to be thread-safe
	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		n = copy(buf[off:], w.Bytes())
	}
	if b.hasJS.Load() {
		buf := b.jsValue.Load().(js.Value)
		buf.Call("set", w, off)
		n = w.Len()
	}
	return n, nil
}

func (b *blob) Grow(off int64) (returnedErr error) {
	defer common.CatchException(&returnedErr)

	// TODO need better consistency if this is to be thread-safe
	newLength := b.length.Load() + off
	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		buf = append(buf, make([]byte, off)...)
		b.bytes.Store(buf)
	}
	if b.hasJS.Load() {
		buf := b.jsValue.Load().(js.Value)
		biggerBuf := uint8Array.New(newLength)
		biggerBuf.Call("set", buf, 0)
		b.jsValue.Store(biggerBuf)
	}
	b.length.Store(newLength)
	return nil
}

func (b *blob) Truncate(size int64) {
	if b.length.Load() < size {
		return
	}

	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		b.bytes.Store(buf[:size])
	}
	if b.hasJS.Load() {
		buf := b.jsValue.Load().(js.Value)
		smallerBuf := buf.Call("slice", 0, size)
		b.jsValue.Store(smallerBuf)
	}
	b.length.Store(size)
}
