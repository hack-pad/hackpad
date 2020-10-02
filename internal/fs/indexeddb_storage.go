package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
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

func (i *indexedDBStorer) readWriteTxnStore() (*indexeddb.ObjectStore, error) {
	txn, err := i.db.Transaction(idbFilesStore, indexeddb.TransactionReadWrite)
	if err != nil {
		return nil, err
	}
	return txn.ObjectStore(idbFilesStore)
}

func (i *indexedDBStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	deleted, err := i.setFile(path, data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if dir == "" || dir == string(filepath.Separator) {
		return nil
	}
	base := filepath.Base(path)
	var parentData storer.FileRecord
	err = i.GetFileRecord(dir, &parentData)
	if err != nil || !parentData.Mode.IsDir() {
		return err
	}
	if deleted {
		parentData.DirNames = removePath(parentData.DirNames, base)
	} else {
		parentData.DirNames = addPath(parentData.DirNames, base)
	}
	return i.SetFileRecord(dir, &parentData)
}

func (i *indexedDBStorer) setFile(path string, data *storer.FileRecord) (deleted bool, err error) {
	if data == nil {
		store, err := i.readWriteTxnStore()
		if err != nil {
			return false, err
		}
		return true, store.Delete(js.ValueOf(path))
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != string(filepath.Separator) {
		var parentData storer.FileRecord
		err := i.GetFileRecord(dir, &parentData)
		if err != nil {
			return false, err
		}
		if !parentData.Mode.IsDir() {
			return false, syscall.ENOTDIR
		}
	}

	store, err := i.readWriteTxnStore()
	if err != nil {
		return false, err
	}
	err = store.Put(js.ValueOf(path), js.ValueOf(map[string]interface{}{
		"Data":     interop.NewByteArray(data.Data),
		"DirNames": interop.SliceFromStrings(data.DirNames),
		"ModTime":  data.ModTime.Unix(),
		"Mode":     uint32(data.Mode),
	}))
	runtime.GC()
	return false, err
}

func removePath(paths []string, path string) []string {
	for i := range paths {
		if paths[i] == path {
			var newPaths []string
			newPaths = append(newPaths, paths[:i]...)
			return append(newPaths, paths[i+1:]...)
		}
	}
	return paths
}

func addPath(paths []string, path string) []string {
	for _, p := range paths {
		if p == path {
			return paths
		}
	}
	paths = append(paths, path)
	sort.Strings(paths)
	return paths
}
