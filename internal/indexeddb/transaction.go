// +build js

package indexeddb

import (
	"syscall/js"
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
	var funcs []js.Func
	newFunc := func(name string) js.Func {
		f := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			for _, fn := range funcs {
				fn.Release()
			}
			return nil
		})
		funcs = append(funcs, f)
		return f
	}
	jsTransaction.Call("addEventListener", "abort", newFunc("abort"))
	jsTransaction.Call("addEventListener", "complete", newFunc("complete"))
	jsTransaction.Call("addEventListener", "error", newFunc("error"))
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
	return &ObjectStore{jsObjectStore: jsObjectStore}, nil
}

func (t *Transaction) Commit() (err error) {
	defer catch(&err)
	t.jsTransaction.Call("commit")
	return nil
}
