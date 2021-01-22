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
