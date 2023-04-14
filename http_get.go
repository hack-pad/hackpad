//go:build js
// +build js

package main

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/promise"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
	"github.com/pkg/errors"
)

var jsFetch = js.Global().Get("fetch")
var uint8Array = js.Global().Get("Uint8Array")

// httpGetFetch sticks to simple calls to the fetch API, then keeps the data inside a JS ArrayBuffer. Memory usage is lower than the "native" http package
func httpGetFetch(path string) (_ blob.Blob, err error) {
	defer common.CatchException(&err)
	prom := jsFetch.Invoke(path)
	resultInt, err := promise.From(prom).Await()
	if err != nil {
		return nil, err
	}
	result := resultInt.(js.Value)

	jsContentType := result.Get("headers").Call("get", "Content-Type")
	if jsContentType.Type() != js.TypeString || jsContentType.String() != "application/wasm" {
		return nil, errors.Errorf("Invalid content type for Wasm: %v", jsContentType)
	}
	body, err := promise.From(result.Call("arrayBuffer")).Await()
	if err != nil {
		return nil, err
	}
	buf := uint8Array.New(body.(js.Value))
	return idbblob.New(buf)
}
