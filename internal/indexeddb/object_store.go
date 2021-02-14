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
	jsObjectStore js.Value
}

func newObjectStore(jsObjectStore js.Value) *ObjectStore {
	return &ObjectStore{jsObjectStore: jsObjectStore}
}

func (o *ObjectStore) Add(key, value js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("add", value, key)), nil
}

func (o *ObjectStore) Clear() (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("clear")), nil
}

func (o *ObjectStore) Count() (_ <-chan int, err error) {
	defer common.CatchException(&err)
	count := make(chan int)
	req := newRequest(o.jsObjectStore.Call("count"))
	req.Listen(func() {
		count <- req.Result().Int()
		close(count)
	}, func() {
		close(count)
	})
	return count, err
}

func (o *ObjectStore) CreateIndex(name string, keyPath js.Value, options IndexOptions) (index *Index, err error) {
	defer common.CatchException(&err)
	jsIndex := o.jsObjectStore.Call("createIndex", name, keyPath, map[string]interface{}{
		"unique":     options.Unique,
		"multiEntry": options.MultiEntry,
	})
	return wrapIndex(jsIndex), nil
}

func (o *ObjectStore) Delete(key js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("delete", key)), nil
}

func (o *ObjectStore) DeleteIndex(name string) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("deleteIndex", name)), nil
}

func (o *ObjectStore) Get(key js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("get", key)), nil
}

func (o *ObjectStore) GetAllKeys(query js.Value) (vals *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("getAllKeys", query)), nil
}

func (o *ObjectStore) GetKey(value js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("getKey", value)), nil
}

func (o *ObjectStore) Index(name string) (index *Index, err error) {
	defer common.CatchException(&err)
	jsIndex := o.jsObjectStore.Call("index", name)
	return wrapIndex(jsIndex), nil
}

func (o *ObjectStore) OpenCursor(key js.Value, direction CursorDirection) (_ <-chan *Cursor, err error) {
	defer common.CatchException(&err)
	cursor := make(chan *Cursor)
	req := newRequest(o.jsObjectStore.Call("openCursor", key, direction.String()))
	req.Listen(func() {
		cursor <- &Cursor{jsCursor: req.Result()}
		close(cursor)
	}, func() {
		close(cursor)
	})
	return cursor, nil
}

/*
func (o *ObjectStore) OpenKeyCursor(keyRange KeyRange, direction CursorDirection) (*Cursor, error) {
	panic("not implemented")
}
*/

func (o *ObjectStore) Put(key, value js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(o.jsObjectStore.Call("put", value, key)), nil
}
