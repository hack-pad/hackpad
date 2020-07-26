package indexeddb

import (
	"syscall/js"
)

type ObjectStoreOptions struct {
	KeyPath       string
	AutoIncrement bool
}

type ObjectStore struct {
	jsObjectStore js.Value
}

func (o *ObjectStore) Add(key, value js.Value) (err error) {
	defer catch(&err)
	_, err = await(processRequest(o.jsObjectStore.Call("add", value, key)))
	return err
}

func (o *ObjectStore) Clear() (err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("clear")
	_, err = await(processRequest(req))
	return err
}

func (o *ObjectStore) Count() (count int, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("count")
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
	return &Index{jsIndex: jsIndex}, nil
}

func (o *ObjectStore) Delete(key js.Value) (err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("delete", key)
	_, err = await(processRequest(req))
	return err
}

func (o *ObjectStore) DeleteIndex(name string) (err error) {
	defer catch(&err)
	o.jsObjectStore.Call("deleteIndex", name)
	return nil
}

func (o *ObjectStore) Get(key js.Value) (val js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("get", key)
	return await(processRequest(req))
}

func (o *ObjectStore) GetKey(value js.Value) (val js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("getKey", value)
	return await(processRequest(req))
}

func (o *ObjectStore) Index(name string) (index *Index, err error) {
	defer catch(&err)
	jsIndex := o.jsObjectStore.Call("index", name)
	return &Index{jsIndex: jsIndex}, nil
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
	req := o.jsObjectStore.Call("put", value, key)
	_, err = await(processRequest(req))
	return err
}
