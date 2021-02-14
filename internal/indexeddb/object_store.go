// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
)

type ObjectStoreOptions struct {
	KeyPath       string
	AutoIncrement bool
}

type ObjectStore struct {
	transaction   *Transaction
	jsObjectStore js.Value
}

func newObjectStore(transaction *Transaction, jsObjectStore js.Value) *ObjectStore {
	return &ObjectStore{transaction: transaction, jsObjectStore: jsObjectStore}
}

func (o *ObjectStore) Add(key, value js.Value) (err error) {
	defer common.CatchException(&err)
	o.jsObjectStore.Call("add", value, key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}

func (o *ObjectStore) Clear() (err error) {
	defer common.CatchException(&err)
	o.jsObjectStore.Call("clear")
	return
}

func (o *ObjectStore) Count() (count int, err error) {
	defer common.CatchException(&err)
	req := newRequest(o.jsObjectStore.Call("count"))
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	jsCount, err := req.Await()
	if err == nil {
		count = jsCount.Int()
	}
	return count, err
}

func (o *ObjectStore) CreateIndex(name string, keyPath js.Value, options IndexOptions) (index *Index, err error) {
	defer common.CatchException(&err)
	jsIndex := o.jsObjectStore.Call("createIndex", name, keyPath, map[string]interface{}{
		"unique":     options.Unique,
		"multiEntry": options.MultiEntry,
	})
	return wrapIndex(o.transaction, jsIndex), nil
}

func (o *ObjectStore) Delete(key js.Value) (err error) {
	defer common.CatchException(&err)
	o.jsObjectStore.Call("delete", key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}

func (o *ObjectStore) DeleteIndex(name string) (err error) {
	defer common.CatchException(&err)
	o.jsObjectStore.Call("deleteIndex", name)
	return nil
}

func (o *ObjectStore) Get(key js.Value) (val js.Value, err error) {
	defer common.CatchException(&err)
	req := newRequest(o.jsObjectStore.Call("get", key))
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return req.Await()
}

func (o *ObjectStore) GetAllKeys(query js.Value) (vals js.Value, err error) {
	defer common.CatchException(&err)
	req := newRequest(o.jsObjectStore.Call("getAllKeys", query))
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return req.Await()
}

func (o *ObjectStore) GetKey(value js.Value) (val js.Value, err error) {
	defer common.CatchException(&err)
	req := newRequest(o.jsObjectStore.Call("getKey", value))
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return req.Await()
}

func (o *ObjectStore) Index(name string) (index *Index, err error) {
	defer common.CatchException(&err)
	jsIndex := o.jsObjectStore.Call("index", name)
	return wrapIndex(o.transaction, jsIndex), nil
}

func (o *ObjectStore) OpenCursor(key js.Value, direction CursorDirection) (cursor *Cursor, err error) {
	defer common.CatchException(&err)
	req := newRequest(o.jsObjectStore.Call("openCursor", key, direction.String()))
	jsCursor, err := req.Await()
	return &Cursor{jsCursor: jsCursor}, err
}

/*
func (o *ObjectStore) OpenKeyCursor(keyRange KeyRange, direction CursorDirection) (*Cursor, error) {
	panic("not implemented")
}
*/

func (o *ObjectStore) Put(key, value js.Value) (err error) {
	defer common.CatchException(&err)
	o.jsObjectStore.Call("put", value, key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}
