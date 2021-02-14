// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
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
	defer common.CatchException(&err)
	jsIndex, err := newRequest(i.jsIndex.Call("count")).Await()
	if err == nil {
		count = jsIndex.Int()
	}
	return count, err
}

func (i *Index) Get(key js.Value) (_ js.Value, err error) {
	defer common.CatchException(&err)
	return newRequest(i.jsIndex.Call("get", key)).Await()
}

func (i *Index) GetKey(value js.Value) (_ js.Value, err error) {
	defer common.CatchException(&err)
	return newRequest(i.jsIndex.Call("getKey", value)).Await()
}

func (i *Index) GetAllKeys(query js.Value) (vals js.Value, err error) {
	defer common.CatchException(&err)
	req := newRequest(i.jsIndex.Call("getAllKeys", query))
	err = i.transaction.Commit()
	if err != nil {
		return
	}
	return req.Await()
}

func (i *Index) OpenCursor(key js.Value, direction CursorDirection) (_ *Cursor, err error) {
	defer common.CatchException(&err)
	req := newRequest(i.jsIndex.Call("openCursor", key, direction.String()))
	jsCursor, err := req.Await()
	return &Cursor{jsCursor: jsCursor}, err
}

//func (i *Index) OpenKeyCursor()
