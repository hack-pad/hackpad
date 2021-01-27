// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
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
	o.transaction.Commit()
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
	o.jsObjectStore.Call("delete", key)
	o.transaction.Commit()
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
	o.jsObjectStore.Call("put", value, key)
	o.transaction.Commit()
	return o.transaction.Await()
}

func BatchGet(objectStore string, key js.Value) func(*Transaction) js.Value {
	return func(txn *Transaction) js.Value {
		o, err := txn.ObjectStore(objectStore)
		if err != nil {
			panic(err)
		}
		return o.jsObjectStore.Call("get", key)
	}
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
) ([]js.Value, error) {
	txn, err := db.Transaction(mode, objectStoreNames...)
	if err != nil {
		return nil, err
	}
	type indexedResult struct {
		int
		js.Value
	}
	results := make(chan indexedResult, len(calls))
	for i, call := range calls {
		index := i
		request := call(txn)
		request.Call("addEventListener", "success", interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			go func() {
				result := request.Get("result")
				results <- indexedResult{index, result}
			}()
			return nil
		}))
	}
	err = txn.Commit()
	if err != nil {
		return nil, err
	}
	err = txn.Await()
	var resultSlice []js.Value
	if err == nil {
		resultSlice = make([]js.Value, len(calls))
		for range resultSlice {
			result := <-results
			resultSlice[result.int] = result.Value
		}
	}
	return resultSlice, err
}
