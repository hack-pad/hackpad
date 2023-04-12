//go:build js
// +build js

package fs

import (
	"syscall/js"
)

type wasmInstancer interface {
	WasmInstance(path string, importObject js.Value) (js.Value, error)
}

func (f *FileDescriptors) WasmInstance(path string, importObject js.Value) (js.Value, error) {
	if instancer, ok := filesystem.(wasmInstancer); ok {
		return instancer.WasmInstance(f.resolvePath(path), importObject)
	}
	panic("Wasm Cache not initialized")
}
