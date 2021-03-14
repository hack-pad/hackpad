// +build js

package main

import (
	"io/ioutil"
	"net/http"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/pkg/errors"
)

var jsFetch = js.Global().Get("fetch")
var uint8Array = js.Global().Get("Uint8Array")

// httpGetGo uses significantly more memory converting from JS to Go, and then back again
func httpGetGo(path string) (blob.Blob, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 { // not success
		return nil, errors.New(resp.Status)
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "application/wasm" {
		return nil, errors.New("Invalid content type for Wasm: " + contentType)
	}
	buf, err := ioutil.ReadAll(resp.Body)
	return blob.NewFromBytes(buf), err
}

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
	return blob.NewFromJS(buf)
}
