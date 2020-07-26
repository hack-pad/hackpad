package fs

import (
	"os"
	"runtime"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/indexeddb"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/storer"
)

const (
	idbVersion    = 1
	idbFilesStore = "files"
)

type IndexedDBFs struct {
	*storer.Fs
}

func NewIndexedDBFs(name string) (_ *IndexedDBFs, err error) {
	db, err := indexeddb.New(name, idbVersion, func(db *indexeddb.DB, oldVersion, newVersion int) error {
		_, err := db.CreateObjectStore(idbFilesStore, indexeddb.ObjectStoreOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	fs := storer.New(&indexedDBStorer{db: db})
	return &IndexedDBFs{fs}, nil
}

type indexedDBStorer struct {
	db *indexeddb.DB
}

func (i *indexedDBStorer) GetFileRecord(path string, dest *storer.FileRecord) error {
	txn, err := i.db.Transaction(idbFilesStore, indexeddb.TransactionReadOnly)
	if err != nil {
		return err
	}
	files, err := txn.ObjectStore(idbFilesStore)
	if err != nil {
		return err
	}
	value, err := files.Get(js.ValueOf(path))
	if value.IsUndefined() {
		return os.ErrNotExist
	}
	if err != nil {
		return err
	}
	runtime.GC()
	jsData := value.Get("Data")
	dest.Data = make([]byte, jsData.Length())
	js.CopyBytesToGo(dest.Data, jsData)
	dest.DirNames = interop.StringsFromJSValue(value.Get("DirNames"))
	dest.ModTime = time.Unix(int64(value.Get("ModTime").Int()), 0)
	dest.Mode = os.FileMode(value.Get("Mode").Int())
	return nil
}

func (i *indexedDBStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	txn, err := i.db.Transaction(idbFilesStore, indexeddb.TransactionReadWrite)
	if err != nil {
		return err
	}
	store, err := txn.ObjectStore(idbFilesStore)
	if err != nil {
		return err
	}

	if data == nil {
		return store.Delete(js.ValueOf(path))
	}
	err = store.Put(js.ValueOf(path), js.ValueOf(map[string]interface{}{
		"Data":     interop.NewByteArray(data.Data),
		"DirNames": interop.SliceFromStrings(data.DirNames),
		"ModTime":  data.ModTime.Unix(),
		"Mode":     uint32(data.Mode),
	}))
	runtime.GC()
	return err
}
