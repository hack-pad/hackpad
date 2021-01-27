// +build js

package indexeddb

import (
	"syscall/js"
)

type IndexOptions struct {
	Unique     bool
	MultiEntry bool
}

type Index struct {
	transaction *Transaction
	jsIndex     js.Value
}

func wrapIndex(txn *Transaction, jsIndex js.Value) *Index {
	return &Index{
		transaction: txn,
		jsIndex:     jsIndex,
	}
}

func (i *Index) Count() (count int, err error) {
	defer catch(&err)
	req := i.jsIndex.Call("count")
	jsIndex, err := await(processRequest(req))
	if err == nil {
		count = jsIndex.Int()
	}
	return count, nil
}

func (i *Index) Get(key js.Value) (_ js.Value, err error) {
	defer catch(&err)
	req := i.jsIndex.Call("get", key)
	return await(processRequest(req))
}

func (i *Index) GetKey(value js.Value) (_ js.Value, err error) {
	defer catch(&err)
	req := i.jsIndex.Call("getKey", value)
	return await(processRequest(req))
}

func (i *Index) GetAllKeys(query js.Value) (vals js.Value, err error) {
	defer catch(&err)
	req := i.jsIndex.Call("getAllKeys", query)
	i.transaction.Commit()
	prom := processRequest(req)
	return await(prom)
}

func (i *Index) OpenCursor(key js.Value, direction CursorDirection) (_ *Cursor, err error) {
	defer catch(&err)
	req := i.jsIndex.Call("openCursor", key, direction.String())
	jsCursor, err := await(processRequest(req))
	return &Cursor{jsCursor: jsCursor}, err
}

//func (i *Index) OpenKeyCursor()
