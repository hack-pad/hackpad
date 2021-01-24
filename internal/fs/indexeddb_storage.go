// +build js

package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/indexeddb"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/log"
	"github.com/spf13/afero"
)

const (
	idbVersion           = 1
	idbFileContentsStore = "contents"
	idbFileInfoStore     = "info"
)

type IndexedDBFs struct {
	*storer.Fs
}

func NewIndexedDBFs(name string) (_ *IndexedDBFs, err error) {
	db, err := indexeddb.New(name, idbVersion, func(db *indexeddb.DB, oldVersion, newVersion int) error {
		_, err := db.CreateObjectStore(idbFileContentsStore, indexeddb.ObjectStoreOptions{})
		if err != nil {
			return err
		}
		_, err = db.CreateObjectStore(idbFileInfoStore, indexeddb.ObjectStoreOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	fs := storer.New(&indexedDBStorer{db: db})
	return &IndexedDBFs{fs}, nil
}

type indexedDBStorer struct {
	db           *indexeddb.DB
	jsPaths      interop.StringCache
	jsProperties interop.StringCache
}

func (i *indexedDBStorer) GetFileRecord(path string, dest *storer.FileRecord) (err error) {
	path = fsutil.NormalizePath(path)
	defer common.CatchException(&err)
	txn, err := i.db.Transaction(indexeddb.TransactionReadOnly, idbFileInfoStore)
	if err != nil {
		return err
	}
	files, err := txn.ObjectStore(idbFileInfoStore)
	if err != nil {
		return err
	}
	log.Debug("Loading file info from JS: ", path)
	value, err := files.Get(i.jsPaths.Value(path))
	return i.extractFileRecord(path, value, err, dest)
}

func (i *indexedDBStorer) extractFileRecord(path string, value js.Value, err error, dest *storer.FileRecord) error {
	if value.IsUndefined() {
		log.Debug("File does not exist: ", path)
		return os.ErrNotExist
	}
	if err != nil {
		log.Debug("Error loading file record: ", path)
		return err
	}
	dest.InitialSize = int64(i.jsProperties.GetProperty(value, "Size").Int())
	dest.DirNames = interop.StringsFromJSValue(i.jsProperties.GetProperty(value, "DirNames"))
	dest.ModTime = time.Unix(int64(i.jsProperties.GetProperty(value, "ModTime").Int()), 0)
	dest.Mode = os.FileMode(i.jsProperties.GetProperty(value, "Mode").Int())
	if dest.Mode.IsDir() {
		log.Debug("Setting no-op directory data fetcher for path ", path)
		dest.DataFn = func() (blob.Blob, error) {
			return blob.NewFromBytes(nil), nil
		}
	} else {
		log.Debug("Setting file data fetcher for path ", path)
		dest.DataFn = i.getFileData(path)
	}
	log.Debug("File loaded: ", path)
	return nil
}

func (i *indexedDBStorer) getFileData(path string) func() (blob.Blob, error) {
	return func() (blob.Blob, error) {
		txn, err := i.db.Transaction(indexeddb.TransactionReadOnly, idbFileContentsStore)
		if err != nil {
			return nil, err
		}
		files, err := txn.ObjectStore(idbFileContentsStore)
		if err != nil {
			return nil, err
		}
		log.Debug("Loading file contents from JS: ", path)
		value, err := files.Get(i.jsPaths.Value(path))
		if value.IsUndefined() {
			return nil, os.ErrNotExist
		}
		if err != nil {
			return nil, err
		}
		return blob.NewFromJS(value)
	}
}

func (i *indexedDBStorer) GetFileRecords(paths []string, dest []*storer.FileRecord) (errs []error) {
	if len(paths) != len(dest) {
		panic(fmt.Sprintf("indexedDBStorer: Paths and dest lengths must be equal: %d != %d", len(paths), len(dest)))
	}
	errs = make([]error, len(paths))
	defer common.CatchException(&errs[0])

	calls := make([]func(*indexeddb.Transaction) js.Value, len(paths))
	for ix := range paths {
		calls[ix] = indexeddb.BatchGet(idbFileInfoStore, i.jsPaths.Value(paths[ix]))
	}

	log.Debug("Loading file infos from JS: ", paths)
	infos, err := i.db.BatchTransaction(indexeddb.TransactionReadOnly, []string{idbFileInfoStore}, calls...)
	if err != nil {
		// error running batch txn, return it in first error slot
		errs[0] = err
		return errs
	}

	for ix := range paths {
		errs[ix] = i.extractFileRecord(paths[ix], infos[ix], nil, dest[ix])
	}
	return errs
}

func (i *indexedDBStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	path = fsutil.NormalizePath(path)
	isRoot := path == "." || path == afero.FilePathSeparator
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
		dir = afero.FilePathSeparator
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
		_, err = i.db.BatchTransaction(
			indexeddb.TransactionReadWrite,
			[]string{idbFileInfoStore, idbFileContentsStore},
			indexeddb.BatchDelete(idbFileInfoStore, i.jsPaths.Value(path)),
			indexeddb.BatchDelete(idbFileContentsStore, i.jsPaths.Value(path)),
		)
		return true, err
	}

	dir := filepath.Dir(path)
	if dir != "" && dir != afero.FilePathSeparator {
		var parentData storer.FileRecord
		err := i.GetFileRecord(dir, &parentData)
		if err != nil {
			return false, err
		}
		if !parentData.Mode.IsDir() {
			return false, syscall.ENOTDIR
		}
	}

	var v []func(*indexeddb.Transaction) js.Value
	if !data.Mode.IsDir() {
		v = append(v, indexeddb.BatchPut(
			idbFileContentsStore,
			i.jsPaths.Value(path), data.Data().JSValue(),
		))
	}
	v = append(v, indexeddb.BatchPut(
		idbFileInfoStore,
		i.jsPaths.Value(path),
		js.ValueOf(map[string]interface{}{
			"DirNames": interop.SliceFromStrings(data.DirNames),
			"ModTime":  data.ModTime.Unix(),
			"Mode":     uint32(data.Mode),
			"Size":     data.Size(),
		}),
	))
	_, err = i.db.BatchTransaction(
		indexeddb.TransactionReadWrite,
		[]string{idbFileContentsStore, idbFileInfoStore},
		v...,
	)
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
