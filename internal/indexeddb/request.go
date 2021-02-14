// +build js

package indexeddb

import (
	"syscall/js"
)

var jsIDBRequest = js.Global().Get("IDBRequest")

type Request struct {
	jsRequest js.Value
}

func newRequest(jsRequest js.Value) *Request {
	if !jsRequest.InstanceOf(jsIDBRequest) {
		panic("Invalid JS request type")
	}
	return &Request{jsRequest}
}

func (r *Request) Await() (result js.Value, err error) {
	done := make(chan struct{})
	r.Listen(func() {
		result = r.Result()
		close(done)
	}, func() {
		err = r.Error()
		close(done)
	})
	<-done
	return
}

func (r *Request) ListenSuccess(success func()) {
	r.Listen(success, nil)
}

func (r *Request) ListenError(failed func()) {
	r.Listen(nil, failed)
}

func (r *Request) Listen(success, failed func()) {
	var errFunc, successFunc js.Func
	// setting up both is required to ensure boath are always released
	errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errFunc.Release()
		successFunc.Release()
		if failed != nil {
			failed()
		}
		return nil
	})
	successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errFunc.Release()
		successFunc.Release()
		if success != nil {
			success()
		}
		return nil
	})
	r.jsRequest.Call("addEventListener", "error", errFunc)
	r.jsRequest.Call("addEventListener", "success", successFunc)
}

func (r *Request) Result() js.Value {
	return r.jsRequest.Get("result")
}

func (r *Request) Error() error {
	jsErr := r.jsRequest.Get("error")
	if jsErr.Truthy() {
		return js.Error{Value: jsErr}
	}
	return nil
}
