//go:build js
// +build js

package interop

import "syscall/js"

func SingleUseFunc(fn func(this js.Value, args []js.Value) interface{}) js.Func {
	var wrapperFn js.Func
	wrapperFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wrapperFn.Release()
		return fn(this, args)
	})
	return wrapperFn
}
