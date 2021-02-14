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
	jsIndex js.Value
}

func wrapIndex(jsIndex js.Value) *Index {
	return &Index{
		jsIndex: jsIndex,
	}
}

func (i *Index) Count() (_ <-chan int, err error) {
	defer common.CatchException(&err)
	count := make(chan int)
	req := newRequest(i.jsIndex.Call("count"))
	req.Listen(func() {
		count <- req.Result().Int()
		close(count)
	}, func() {
		close(count)
	})
	return count, nil
}

func (i *Index) Get(key js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(i.jsIndex.Call("get", key)), nil
}

func (i *Index) GetKey(value js.Value) (_ *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(i.jsIndex.Call("getKey", value)), nil
}

func (i *Index) GetAllKeys(query js.Value) (vals *Request, err error) {
	defer common.CatchException(&err)
	return newRequest(i.jsIndex.Call("getAllKeys", query)), nil
}

func (i *Index) OpenCursor(key js.Value, direction CursorDirection) (_ <-chan *Cursor, err error) {
	defer common.CatchException(&err)
	cursor := make(chan *Cursor)
	req := newRequest(i.jsIndex.Call("openCursor", key, direction.String()))
	req.Listen(func() {
		cursor <- &Cursor{jsCursor: req.Result()}
		close(cursor)
	}, func() {
		close(cursor)
	})
	return cursor, nil
}

//func (i *Index) OpenKeyCursor()
