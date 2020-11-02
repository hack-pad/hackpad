// +build js

package common

import "syscall/js"

func (f FID) JSValue() js.Value {
	return js.ValueOf(uint64(f))
}

func (p PID) JSValue() js.Value {
	return js.ValueOf(uint64(p))
}
