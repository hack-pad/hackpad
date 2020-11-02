// +build js

package process

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
)

var (
	jsGo   = js.Global().Get("Go")
	jsWasm = js.Global().Get("WebAssembly")
)

func (p *process) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"pid":   p.pid,
		"ppid":  p.parentPID,
		"error": interop.WrapAsJSError(p.err, "spawn"),
	})
}

func (p *process) StartCPUProfile() error {
	return interop.StartCPUProfile(p.ctx)
}
