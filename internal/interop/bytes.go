// +build js

package interop

import "syscall/js"

var uint8Array = js.Global().Get("Uint8Array")

func NewByteArray(b []byte) js.Value {
	buf := uint8Array.New(len(b))
	js.CopyBytesToJS(buf, b)
	return buf
}
