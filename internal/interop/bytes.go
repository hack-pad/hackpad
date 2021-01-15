// +build js

package interop

import (
	"syscall/js"

	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var uint8Array = js.Global().Get("Uint8Array")

func NewByteArray(b []byte) js.Value {
	buf := uint8Array.New(len(b))
	js.CopyBytesToJS(buf, b)
	return buf
}

type BlobFile interface {
	ReadBlobAt(length int, off int64) (blob Blob, n int, err error)
	WriteBlobAt(p Blob, off int64) (n int, err error)
}

type Blob interface {
	js.Wrapper
	Bytes() []byte
	Len() int
	Slice(start, end int64) (Blob, error)
	Set(w Blob, off int64) (n int, err error)
	Grow(off int64) error
}

type blob struct {
	bytes    atomic.Value
	hasBytes atomic.Bool
	jsValue  atomic.Value
	hasJS    atomic.Bool
	length   atomic.Int64
}

func NewBlobBytes(buf []byte) Blob {
	b := &blob{}
	b.hasBytes.Store(true)
	b.bytes.Store(buf)
	b.length.Store(int64(len(buf)))
	return b
}

func NewBlobJS(buf js.Value) (Blob, error) {
	if !buf.Truthy() {
		return NewBlobBytes(nil), nil
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
	jsBuf := NewByteArray(buf)
	b.jsValue.Store(jsBuf)
	b.hasJS.Store(true)
	return jsBuf
}

func (b *blob) Len() int {
	return int(b.length.Load())
}

func (b *blob) Slice(start, end int64) (_ Blob, returnedErr error) {
	defer CatchException(&returnedErr)

	if start == 0 && end == b.length.Load() {
		return b, nil
	}
	if b.hasBytes.Load() {
		buf := b.bytes.Load().([]byte)
		return NewBlobBytes(buf[start:end]), nil
	}
	buf := b.jsValue.Load().(js.Value)
	return NewBlobJS(buf.Call("slice", start, end))
}

func (b *blob) Set(w Blob, off int64) (n int, returnedErr error) {
	defer CatchException(&returnedErr)

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
	defer CatchException(&returnedErr)

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
