//go:build js
// +build js

package process

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
)

var (
	jsGo = js.Global().Get("Go")
)

func (p *process) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"pid":   p.pid.JSValue(),
		"ppid":  p.parentPID.JSValue(),
		"error": interop.WrapAsJSError(p.err, "spawn"),
	})
}

func (p *process) StartCPUProfile() error {
	return interop.StartCPUProfile(p.ctx)
}
