// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
)

type TransactionMode int

const (
	TransactionReadOnly TransactionMode = iota
	TransactionReadWrite
)

func (m TransactionMode) String() string {
	switch m {
	case TransactionReadWrite:
		return "readwrite"
	default:
		return "readonly"
	}
}

type Transaction struct {
	jsTransaction js.Value
}

func wrapTransaction(jsTransaction js.Value) *Transaction {
	return &Transaction{jsTransaction: jsTransaction}
}

func (t *Transaction) Abort() (err error) {
	defer catch(&err)
	t.jsTransaction.Call("abort")
	return nil
}

func (t *Transaction) ObjectStore(name string) (_ *ObjectStore, err error) {
	defer catch(&err)
	jsObjectStore := t.jsTransaction.Call("objectStore", name)
	return newObjectStore(t, jsObjectStore), nil
}

func (t *Transaction) Commit() (err error) {
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
		errFunc.Release()
		completeFunc.Release()
		reject(err)
		return nil
	})
	completeFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		errFunc.Release()
		completeFunc.Release()
		resolve(nil)
		return nil
	})
	t.jsTransaction.Call("addEventListener", "error", errFunc)
	t.jsTransaction.Call("addEventListener", "complete", completeFunc)
	return prom
}
