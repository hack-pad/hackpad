package promise

import (
	"runtime/debug"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
)

var jsPromise = js.Global().Get("Promise")

type Promise struct {
	value js.Value
}

type Resolver func(interface{})

func From(promiseValue js.Value) Promise {
	return Promise{value: promiseValue}
}

func New() (resolve, reject Resolver, promise Promise) {
	resolvers := make(chan Resolver, 2)
	promise = From(
		jsPromise.New(interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			resolve, reject := args[0], args[1]
			resolvers <- func(result interface{}) { resolve.Invoke(result) }
			resolvers <- func(result interface{}) { reject.Invoke(result) }
			return nil
		})),
	)
	resolve, reject = <-resolvers, <-resolvers
	return
}

func (p Promise) Then(fn func(value js.Value) interface{}) Promise {
	return p.do("then", fn)
}

func (p Promise) do(methodName string, fn func(value js.Value) interface{}) Promise {
	return Promise{
		value: p.value.Call(methodName, interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			var value js.Value
			if len(args) > 0 {
				value = args[0]
			}
			return fn(value)
		})),
	}
}

func (p Promise) Catch(fn func(rejectedReason js.Value) interface{}) Promise {
	stack := string(debug.Stack())
	return p.do("catch", func(rejectedReason js.Value) interface{} {
		log.ErrorJSValues(
			js.ValueOf("Promise rejected:"),
			rejectedReason,
			js.ValueOf(stack),
		)
		return fn(rejectedReason)
	})
}

func (p Promise) JSValue() js.Value {
	return p.value
}
