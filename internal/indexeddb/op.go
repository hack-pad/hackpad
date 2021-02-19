// +build js

package indexeddb

import (
	"syscall/js"
)

type Op = func(*Transaction) (*Request, error)

func GetOp(objectStore string, key js.Value) Op {
	return func(txn *Transaction) (*Request, error) {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			return nil, err
		}
		return o.Get(key)
	}
}

func PutOp(objectStore string, key, value js.Value) Op {
	return func(txn *Transaction) (*Request, error) {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			return nil, err
		}
		return o.Put(key, value)
	}
}

func DeleteOp(objectStore string, key js.Value) Op {
	return func(txn *Transaction) (*Request, error) {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			return nil, err
		}
		return o.Delete(key)
	}
}
