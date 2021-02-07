// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
)

var supportsTransactionCommit = js.Global().Get("IDBTransaction").Get("prototype").Get("commit").Truthy()

type TransactionMode int

const (
	TransactionReadOnly TransactionMode = iota
	TransactionReadWrite
)

var modeCache interop.StringCache

func (m TransactionMode) String() string {
	switch m {
	case TransactionReadWrite:
		return "readwrite"
	default:
		return "readonly"
	}
}

func (m TransactionMode) JSValue() js.Value {
	return modeCache.Value(m.String())
}

type Transaction struct {
	jsTransaction  js.Value
	jsObjectStores map[string]*ObjectStore
}

func wrapTransaction(jsTransaction js.Value) *Transaction {
	return &Transaction{
		jsTransaction:  jsTransaction,
		jsObjectStores: make(map[string]*ObjectStore),
	}
}

func (t *Transaction) Abort() (err error) {
	defer catch(&err)
	t.jsTransaction.Call("abort")
	return nil
}

func (t *Transaction) ObjectStore(name string) (_ *ObjectStore, err error) {
	if store, ok := t.jsObjectStores[name]; ok {
		return store, nil
	}
	defer catch(&err)
	jsObjectStore := t.jsTransaction.Call("objectStore", name)
	store := newObjectStore(t, jsObjectStore)
	t.jsObjectStores[name] = store
	return store, nil
}

func (t *Transaction) Commit() (err error) {
	if !supportsTransactionCommit {
		return nil
	}

	defer catch(&err)
	t.jsTransaction.Call("commit")
	return nil
}

func (t *Transaction) Await() error {
	_, err := t.prepareAwait().Await()
	return err
}

func (t *Transaction) prepareAwait() promise.Promise {
	resolve, reject, prom := promise.NewGo()

	var errFunc, completeFunc js.Func
	errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := t.jsTransaction.Get("error")
		t.jsTransaction.Call("abort")
		go func() {
			errFunc.Release()
			completeFunc.Release()
			reject(err)
		}()
		return nil
	})
	completeFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			errFunc.Release()
			completeFunc.Release()
			resolve(nil)
		}()
		return nil
	})
	t.jsTransaction.Call("addEventListener", "error", errFunc)
	t.jsTransaction.Call("addEventListener", "complete", completeFunc)
	return prom
}
