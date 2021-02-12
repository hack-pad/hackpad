// +build js

package indexeddb

import (
	"syscall/js"
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
	defer catch(&err)
	o.jsObjectStore.Call("add", value, key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}

func (o *ObjectStore) Clear() (err error) {
	defer catch(&err)
	o.jsObjectStore.Call("clear")
	return o.transaction.Await()
}

func (o *ObjectStore) Count() (count int, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("count")
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	jsCount, err := await(processRequest(req))
	if err == nil {
		count = jsCount.Int()
	}
	return count, err
}

func (o *ObjectStore) CreateIndex(name string, keyPath js.Value, options IndexOptions) (index *Index, err error) {
	defer catch(&err)
	jsIndex := o.jsObjectStore.Call("createIndex", name, keyPath, map[string]interface{}{
		"unique":     options.Unique,
		"multiEntry": options.MultiEntry,
	})
	return wrapIndex(o.transaction, jsIndex), nil
}

func (o *ObjectStore) Delete(key js.Value) (err error) {
	defer catch(&err)
	o.jsObjectStore.Call("delete", key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}

func (o *ObjectStore) DeleteIndex(name string) (err error) {
	defer catch(&err)
	o.jsObjectStore.Call("deleteIndex", name)
	return nil
}

func (o *ObjectStore) Get(key js.Value) (val js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("get", key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	prom := processRequest(req)
	return await(prom)
}

func (o *ObjectStore) GetAllKeys(query js.Value) (vals js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("getAllKeys", query)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	prom := processRequest(req)
	return await(prom)
}

func (o *ObjectStore) GetKey(value js.Value) (val js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("getKey", value)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return await(processRequest(req))
}

func (o *ObjectStore) Index(name string) (index *Index, err error) {
	defer catch(&err)
	jsIndex := o.jsObjectStore.Call("index", name)
	return wrapIndex(o.transaction, jsIndex), nil
}

func (o *ObjectStore) OpenCursor(key js.Value, direction CursorDirection) (cursor *Cursor, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("openCursor", key, direction.String())
	jsCursor, err := await(processRequest(req))
	return &Cursor{jsCursor: jsCursor}, err
}

/*
func (o *ObjectStore) OpenKeyCursor(keyRange KeyRange, direction CursorDirection) (*Cursor, error) {
	panic("not implemented")
}
*/

func (o *ObjectStore) Put(key, value js.Value) (err error) {
	defer catch(&err)
	o.jsObjectStore.Call("put", value, key)
	err = o.transaction.Commit()
	if err != nil {
		return
	}
	return o.transaction.Await()
}
