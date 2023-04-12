//go:build js
// +build js

package promise

import (
	"runtime/debug"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
)

var jsPromise = js.Global().Get("Promise")

type JS struct {
	value js.Value
}

func From(promiseValue js.Value) JS {
	return JS{value: promiseValue}
}

func New() (resolve, reject Resolver, promise JS) {
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

func (p JS) Then(fn func(value interface{}) interface{}) Promise {
	return p.do("then", fn)
}

func (p JS) do(methodName string, fn func(value interface{}) interface{}) Promise {
	return JS{
		value: p.value.Call(methodName, interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			var value js.Value
			if len(args) > 0 {
				value = args[0]
			}
			return fn(value)
		})),
	}
}

func (p JS) Catch(fn func(rejectedReason interface{}) interface{}) Promise {
	stack := string(debug.Stack())
	return p.do("catch", func(rejectedReason interface{}) interface{} {
		log.ErrorJSValues(
			js.ValueOf("Promise rejected:"),
			rejectedReason,
			js.ValueOf(stack),
		)
		return fn(rejectedReason)
	})
}

func (p JS) Await() (interface{}, error) {
	errs := make(chan error, 1)
	results := make(chan js.Value, 1)
	p.Then(func(value interface{}) interface{} {
		results <- value.(js.Value)
		close(results)
		return nil
	}).Catch(func(rejectedReason interface{}) interface{} {
		err := js.Error{Value: rejectedReason.(js.Value)}
		errs <- err
		close(errs)
		return nil
	})
	select {
	case err := <-errs:
		return js.Null(), err
	case result := <-results:
		return result, nil
	}
}

func (p JS) JSValue() js.Value {
	return p.value
}
