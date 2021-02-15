// +build js

package nativeio

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/promise"
)

var (
	sharedArrayBuffer = js.Global().Get("SharedArrayBuffer")
	uint8Array        = js.Global().Get("Uint8Array")
)

type File struct {
	name   string
	jsFile js.Value
}

func newFile(name string, jsFile js.Value) *File {
	return &File{
		name:   name,
		jsFile: jsFile,
	}
}

func (f *File) Close() (err error) {
	defer common.CatchException(&err)
	_, err = promise.From(f.jsFile.Call("close")).Await()
	return err
}

func (f *File) Flush() (err error) {
	defer common.CatchException(&err)
	_, err = promise.From(f.jsFile.Call("flush")).Await()
	return err
}

func (f *File) Size() (size uint64, err error) {
	defer common.CatchException(&err)
	jsLength, err := promise.From(f.jsFile.Call("getLength")).Await()
	if err == nil {
		size = uint64(jsLength.(js.Value).Int())
	}
	return
}

func (f *File) Truncate(size uint64) (err error) {
	defer common.CatchException(&err)
	_, err = promise.From(f.jsFile.Call("setLength", size)).Await()
	return err
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	b, n, err := f.ReadBlobAt(len(p), off)
	if err == nil {
		n = js.CopyBytesToGo(p, b.JSValue())
	}
	return
}

func newSharedArray(length int) blob.Blob {
	sharedArray := sharedArrayBuffer.New(length)
	array := uint8Array.New(sharedArray)
	b, err := blob.NewFromJS(array)
	if err != nil {
		panic(err)
	}
	return b
}

func (f *File) ReadBlobAt(length int, off int64) (b blob.Blob, n int, err error) {
	b = newSharedArray(length)
	jsN, err := promise.From(f.jsFile.Call("read", b, off)).Await()
	if err == nil {
		n = jsN.(js.Value).Int()
	}
	return
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	return f.WriteBlobAt(blob.NewFromBytes(p), off)
}

func (f *File) WriteBlobAt(p blob.Blob, off int64) (n int, err error) {
	sharedArray := newSharedArray(p.Len())
	_, err = sharedArray.Set(p, 0)
	if err != nil {
		return
	}
	p = sharedArray
	jsN, err := promise.From(f.jsFile.Call("write", p, off)).Await()
	if err == nil {
		n = jsN.(js.Value).Int()
	}
	return
}
