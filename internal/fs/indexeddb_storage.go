// +build js

package fs

import (
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/indexeddb"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/log"
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
	db        *indexeddb.DB
	jsStrings interop.StringCache
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
	value, err := files.Get(i.jsStrings.Value(path))
	if value.IsUndefined() {
		return os.ErrNotExist
	}
	if err != nil {
		return err
	}
	log.Debug("Loading file from JS: ", path)
	jsData := value.Get("Data")
	dest.Data, err = blob.NewFromJS(jsData)
	if err != nil {
		return err
	}
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
	path = filepath.Clean(path)
	isRoot := path == "." || path == string(filepath.Separator)
	if data == nil && isRoot {
		return syscall.ENOSYS // cannot delete root dir
	}
	deleted, err := i.setFile(path, data)
	if err != nil {
		return err
	}

	// update parent dir's entries
	if isRoot {
		return nil // root directory does not have a parent dir
	}
	dir := filepath.Dir(path)
	if dir == "." {
		dir = string(filepath.Separator)
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
	_, err = i.setFile(dir, &parentData)
	return err
}

func (i *indexedDBStorer) setFile(path string, data *storer.FileRecord) (deleted bool, err error) {
	if data == nil {
		store, err := i.readWriteTxnStore()
		if err != nil {
			return false, err
		}
		return true, store.Delete(i.jsStrings.Value(path))
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
	err = store.Put(i.jsStrings.Value(path), js.ValueOf(map[string]interface{}{
		"Data":     data.Data,
		"DirNames": interop.SliceFromStrings(data.DirNames),
		"ModTime":  data.ModTime.Unix(),
		"Mode":     uint32(data.Mode),
	}))
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
