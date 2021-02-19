// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
)

var (
	jsIndexedDB = js.Global().Get("indexedDB")
)

type DB struct {
	jsDB    js.Value
	jsFuncs interop.CallCache
}

func DeleteDatabase(name string) error {
	_, err := newRequest(jsIndexedDB.Call("deleteDatabase", name)).Await()
	return err
}

func New(name string, version int, upgrader func(db *DB, oldVersion, newVersion int) error) (*DB, error) {
	db := &DB{}
	jsRequest := jsIndexedDB.Call("open", name, version)
	request := newRequest(jsRequest)

	request.ListenSuccess(func() {
		jsDB, err := request.Result()
		if err != nil {
			panic(err)
		}
		jsDB.Call("addEventListener", "versionchange", js.FuncOf(func(js.Value, []js.Value) interface{} {
			log.Print("Version change detected, closing DB...")
			jsDB.Call("close")
			return nil
		}))
		logEvent := func(name string) js.Func {
			return js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				log.Warn("Event: ", name)
				log.WarnJSValues(interop.SliceFromJSValues(args)...)
				return nil
			})
		}
		jsDB.Call("addEventListener", "error", logEvent("error"))
		jsDB.Call("addEventListener", "abort", logEvent("abort"))
		jsDB.Call("addEventListener", "close", logEvent("close"))
	})
	jsRequest.Call("addEventListener", "upgradeneeded", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		var err error
		db.jsDB, err = request.Result()
		if err != nil {
			panic(err)
		}
		err = upgrader(db, event.Get("oldVersion").Int(), event.Get("newVersion").Int())
		if err != nil {
			panic(err)
		}
		return nil
	}))
	var err error
	db.jsDB, err = request.Await()
	return db, err
}

func (db *DB) CreateObjectStore(name string, options ObjectStoreOptions) (_ *ObjectStore, err error) {
	defer common.CatchException(&err)
	jsOptions := map[string]interface{}{
		"autoIncrement": options.AutoIncrement,
	}
	if options.KeyPath != "" {
		jsOptions["keyPath"] = options.KeyPath
	}
	jsObjectStore := db.jsDB.Call("createObjectStore", name, jsOptions)
	return newObjectStore(jsObjectStore), nil
}

func (db *DB) DeleteObjectStore(name string) (err error) {
	defer common.CatchException(&err)
	db.jsDB.Call("deleteObjectStore", name)
	return nil
}

func (db *DB) Close() (err error) {
	defer common.CatchException(&err)
	db.jsDB.Call("close")
	return nil
}

func (db *DB) Transaction(mode TransactionMode, objectStoreNames ...string) (_ *Transaction, err error) {
	defer common.CatchException(&err)
	jsTxn := db.jsFuncs.Call(db.jsDB, "transaction", interop.SliceFromStrings(objectStoreNames), mode)
	return wrapTransaction(jsTxn), nil
}
