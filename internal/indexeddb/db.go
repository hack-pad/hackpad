package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/promise"
)

var (
	jsIndexedDB = js.Global().Get("indexedDB")
)

type DB struct {
	jsDB js.Value
}

func DeleteDatabase(name string) error {
	prom := processRequest(jsIndexedDB.Call("deleteDatabase", name))
	_, err := await(prom)
	return err
}

func New(name string, version int, upgrader func(db *DB, oldVersion, newVersion int) error) (*DB, error) {
	db := &DB{}
	request := jsIndexedDB.Call("open", name, version)

	resolve, reject, prom := promise.NewGoPromise()
	request.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reject(request.Get("error"))
		return nil
	}))
	request.Call("addEventListener", "success", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resolve(request.Get("result"))
		return nil
	}))
	request.Call("addEventListener", "upgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		db.jsDB = request.Get("result")
		err := upgrader(db, event.Get("oldVersion").Int(), event.Get("newVersion").Int())
		if err != nil {
			reject(err)
		}
		return nil
	}))
	var err error
	db.jsDB, err = await(prom)
	return db, err
}

func (db *DB) CreateObjectStore(name string, options ObjectStoreOptions) (_ *ObjectStore, err error) {
	defer catch(&err)
	jsOptions := map[string]interface{}{
		"autoIncrement": options.AutoIncrement,
	}
	if options.KeyPath != "" {
		jsOptions["keyPath"] = options.KeyPath
	}
	jsObjectStore := db.jsDB.Call("createObjectStore", name, jsOptions)
	return &ObjectStore{jsObjectStore: jsObjectStore}, nil
}

func (db *DB) DeleteObjectStore(name string) (err error) {
	defer catch(&err)
	db.jsDB.Call("deleteObjectStore", name)
	return nil
}

func (db *DB) Close() (err error) {
	defer catch(&err)
	db.jsDB.Call("close")
	return nil
}

func (db *DB) Transaction(objectStoreName string, mode TransactionMode) (_ *Transaction, err error) {
	defer catch(&err)
	jsTxn := db.jsDB.Call("transaction", objectStoreName, mode.String())
	return &Transaction{jsTransaction: jsTxn}, nil
}
