package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

func processRequest(request js.Value) promise.Promise {
	resolve, reject, prom := promise.New()
	request.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reject(request.Get("error"))
		return nil
	}))
	request.Set("onsuccess", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve(request.Get("result"))
		return nil
	}))
	return prom
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
