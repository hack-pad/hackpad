// +build js

package indexeddb

import "syscall/js"

type Op = func(*Transaction) *Request

func GetOp(objectStore string, key js.Value) Op {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return newRequest(o.jsObjectStore.Call("get", key))
	}
}

func PutOp(objectStore string, key, value js.Value) Op {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return newRequest(o.jsObjectStore.Call("put", value, key))
	}
}

func DeleteOp(objectStore string, key js.Value) Op {
	return func(txn *Transaction) *Request {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return newRequest(o.jsObjectStore.Call("delete", key))
	}
}
