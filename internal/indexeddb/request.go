// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

type Request struct {
	jsRequest js.Value
}

func (r *Request) Await() (js.Value, error) {
	return await(processRequest(r.jsRequest))
}

func (r *Request) ListenSuccess(success func()) {
	var errFunc, successFunc js.Func
	errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// errFunc setup is required to ensure successFunc is always released
		errFunc.Release()
		successFunc.Release()
		return nil
	})
	successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errFunc.Release()
		successFunc.Release()
		success()
		return nil
	})
	r.jsRequest.Call("addEventListener", "error", errFunc)
	r.jsRequest.Call("addEventListener", "success", successFunc)
}

func (r *Request) Result() js.Value {
	return r.jsRequest.Get("result")
}

func BatchGet(objectStore string, key js.Value) func(*Transaction) *Request {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return &Request{o.jsObjectStore.Call("get", key)}
	}
}

func BatchPut(objectStore string, key, value js.Value) func(*Transaction) *Request {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return &Request{o.jsObjectStore.Call("put", value, key)}
	}
}

func BatchDelete(objectStore string, key js.Value) func(*Transaction) *Request {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return &Request{o.jsObjectStore.Call("delete", key)}
	}
}

func processRequest(request js.Value) promise.Promise {
	resolve, reject, prom := promise.NewGo()

	var errFunc, successFunc js.Func
	errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := request.Get("error")
		go func() {
			errFunc.Release()
			successFunc.Release()
			reject(err)
		}()
		return nil
	})
	successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		result := request.Get("result")
		go func() {
			errFunc.Release()
			successFunc.Release()
			resolve(result)
		}()
		return nil
	})
	request.Call("addEventListener", "error", errFunc)
	request.Call("addEventListener", "success", successFunc)
	return prom
}

func await(prom promise.Promise) (js.Value, error) {
	val, err := prom.Await()
	if err != nil {
		return js.Value{}, err
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
