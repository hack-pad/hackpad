// +build js

package jsfunc

import "syscall/js"

type Func = func(this js.Value, args []js.Value) interface{}

func SingleUse(fn Func) js.Func {
	var wrapperFn js.Func
	wrapperFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wrapperFn.Release()
		return fn(this, args)
	})
	return wrapperFn
}
