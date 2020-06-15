package promise

import (
	"syscall/js"
)

type Promise struct {
	value js.Value
}

func New(promiseValue js.Value) Promise {
	return Promise{value: promiseValue}
}

func (p Promise) Then(fn func(value js.Value) interface{}) Promise {
	return p.do("then", fn)
}

func (p Promise) do(methodName string, fn func(value js.Value) interface{}) Promise {
	return Promise{
		value: p.value.Call(methodName, singleUseFunc(func(this js.Value, args []js.Value) interface{} {
			var value js.Value
			if len(args) > 0 {
				value = args[0]
			}
			return fn(value)
		})),
	}
}

func (p Promise) Catch(fn func(rejectedReason js.Value) interface{}) Promise {
	return p.do("catch", fn)
}

func (p Promise) JSValue() js.Value {
	return p.value
}

func singleUseFunc(fn func(this js.Value, args []js.Value) interface{}) js.Func {
	var wrapperFn js.Func
	wrapperFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		wrapperFn.Release()
		return fn(this, args)
	})
	return wrapperFn
}
