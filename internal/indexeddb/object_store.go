// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
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
	req := o.jsObjectStore.Call("add", value, key)
	o.transaction.Commit()
	_, err = await(processRequest(req))
	return err
}

func (o *ObjectStore) Clear() (err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("clear")
	o.transaction.Commit()
	_, err = await(processRequest(req))
	return err
}

func (o *ObjectStore) Count() (count int, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("count")
	o.transaction.Commit()
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
	o.transaction.Commit()
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
	o.transaction.Commit()
	prom := processRequest(req)
	return await(prom)
}

func (o *ObjectStore) GetKey(value js.Value) (val js.Value, err error) {
	defer catch(&err)
	req := o.jsObjectStore.Call("getKey", value)
	o.transaction.Commit()
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
	o.transaction.Commit()
	_, err = await(processRequest(req))
	return err
}

func BatchPut(objectStore string, key, value js.Value) func(*Transaction) js.Value {
	return func(txn *Transaction) js.Value {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return o.jsObjectStore.Call("put", value, key)
	}
}

func BatchDelete(objectStore string, key js.Value) func(*Transaction) js.Value {
	return func(txn *Transaction) js.Value {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return o.jsObjectStore.Call("delete", key)
	}
}

func (db *DB) BatchTransaction(
	mode TransactionMode,
	objectStoreNames []string,
	calls ...func(*Transaction) (request js.Value),
) (interface{}, error) {
	txn, err := db.Transaction(mode, objectStoreNames...)
	if err != nil {
		return nil, err
	}
	resolve, reject, prom := promise.NewGo()
	fn := func(result js.Value) { resolve(result) }
	for i := len(calls) - 1; i >= 0; i-- {
		prevFn := fn
		call := calls[i]
		lastCall := i == len(calls)-1
		fn = func(_ js.Value) {
			request := call(txn)
			if lastCall {
				txn.Commit()
			}

			var errFunc, successFunc js.Func
			errFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errFunc.Release()
				successFunc.Release()
				txn.jsTransaction.Call("abort")
				err := js.Error{Value: request.Get("error")}
				log.Error("Error batching: ", err)
				reject(err)
				return nil
			})
			successFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errFunc.Release()
				successFunc.Release()
				prevFn(request.Get("result"))
				return nil
			})
			request.Call("addEventListener", "error", errFunc)
			request.Call("addEventListener", "success", successFunc)
		}
	}
	fn(js.Null())
	return prom.Await()
}
