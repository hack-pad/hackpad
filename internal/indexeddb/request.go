package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

func processRequest(request js.Value) promise.GoPromise {
	resolve, reject, prom := promise.NewGoPromise()
	request.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reject(request.Get("error"))
		return nil
	}))
	request.Call("addEventListener", "success", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve(request.Get("result"))
		return nil
	}))
	return prom
}

func await(prom promise.GoPromise) (js.Value, error) {
	val, errVal := promise.AwaitGo(prom)
	if errVal != nil {
		return js.Value{}, js.Error{Value: errVal.(js.Value)}
	}
	return val.(js.Value), nil
}

func catch(err *error) {
	r := recover()
	if r == nil {
		return
	}
	switch val := r.(type) {
	case error:
		*err = val
	case js.Value:
		*err = js.Error{Value: val}
	default:
		*err = errors.Errorf("%+v", val)
	}
}
