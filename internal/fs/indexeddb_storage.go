// +build js

package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/indexeddb"
	"github.com/johnstarich/go-wasm/internal/indexeddb/queue"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/log"
	"github.com/spf13/afero"
)

const (
	idbVersion           = 1
	idbFileContentsStore = "contents"
	idbFileInfoStore     = "info"
	idbParentKey         = "Parent"
)

const (
	maxSetQueue      = 64
	setQueueInterval = 20 * time.Millisecond
)

type IndexedDBFs struct {
	*storer.Fs
}

func newPersistDB(name string) (*IndexedDBFs, error) {
	// TODO support Chromium nativeIO
	return NewIndexedDBFs(name)
}

func NewIndexedDBFs(name string) (_ *IndexedDBFs, err error) {
	db, err := indexeddb.New(name, idbVersion, func(db *indexeddb.DB, oldVersion, newVersion int) error {
		_, err := db.CreateObjectStore(idbFileContentsStore, indexeddb.ObjectStoreOptions{})
		if err != nil {
			return err
		}
		infos, err := db.CreateObjectStore(idbFileInfoStore, indexeddb.ObjectStoreOptions{})
		if err != nil {
			return err
		}
		_, err = infos.CreateIndex(idbParentKey, js.ValueOf(idbParentKey), indexeddb.IndexOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	setQueue := queue.New(maxSetQueue)
	setQueue.StartAsync(context.Background(), setQueueInterval, db)
	fs := storer.New(&indexedDBStorer{
		db:       db,
		setQueue: setQueue,
	})
	return &IndexedDBFs{fs}, nil
}

func (i *IndexedDBFs) Clear() error {
	db := i.Storer.(*indexedDBStorer).db
	stores := []string{idbFileContentsStore, idbFileInfoStore}
	txn, err := db.Transaction(indexeddb.TransactionReadWrite, stores...)
	if err != nil {
		return err
	}
	for _, name := range stores {
		store, err := txn.ObjectStore(name)
		if err != nil {
			return err
		}
		_, err = store.Clear()
		if err != nil {
			return err
		}
	}
	err = txn.Commit()
	if err != nil {
		return err
	}
	return txn.Await()
}

type indexedDBStorer struct {
	db           *indexeddb.DB
	jsPaths      interop.StringCache
	jsProperties interop.StringCache
	setQueue     *queue.Queue
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
	req, err := files.Get(i.jsPaths.Value(path))
	if err != nil {
		return err
	}
	value, err := req.Await()
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
	dest.ModTime = time.Unix(int64(i.jsProperties.GetProperty(value, "ModTime").Int()), 0)
	dest.Mode = i.getMode(value)
	if dest.Mode.IsDir() {
		log.Debug("Setting directory data fetchers for path ", path)
		dest.DataFn = func() (blob.Blob, error) {
			return blob.NewFromBytes(nil), nil
		}
		dest.DirNamesFn = i.getDirNames(path)
	} else {
		log.Debug("Setting file data fetchers for path ", path)
		dest.DataFn = i.getFileData(path)
		dest.DirNamesFn = func() ([]string, error) {
			return nil, nil
		}
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
		req, err := files.Get(i.jsPaths.Value(path))
		if err != nil {
			return nil, err
		}
		value, err := req.Await()
		if value.IsUndefined() {
			return nil, os.ErrNotExist
		}
		if err != nil {
			return nil, err
		}
		return blob.NewFromJS(value)
	}
}

func (i *indexedDBStorer) getDirNames(path string) func() ([]string, error) {
	return func() (_ []string, err error) {
		defer func() {
			common.CatchException(&err)
			if err != nil {
				log.Error(err, "\n", string(debug.Stack()))
			}
		}()
		txn, err := i.db.Transaction(indexeddb.TransactionReadOnly, idbFileInfoStore)
		if err != nil {
			return nil, err
		}
		files, err := txn.ObjectStore(idbFileInfoStore)
		if err != nil {
			return nil, err
		}

		parentIndex, err := files.Index(idbParentKey)
		if err != nil {
			return nil, err
		}
		keysReq, err := parentIndex.GetAllKeys(i.jsPaths.Value(path))
		if err != nil {
			return nil, err
		}
		jsKeys, err := keysReq.Await()
		var keys []string
		if err == nil {
			keys = interop.StringsFromJSValue(jsKeys)
			for i := range keys {
				keys[i] = filepath.Base(keys[i])
			}
		}
		return keys, err
	}
}

func (i *indexedDBStorer) getMode(fileRecord js.Value) os.FileMode {
	mode := i.jsProperties.GetProperty(fileRecord, "Mode")
	return os.FileMode(mode.Int())
}

func (i *indexedDBStorer) GetFileRecords(paths []string, dest []*storer.FileRecord) (errs []error) {
	if len(paths) != len(dest) {
		panic(fmt.Sprintf("indexedDBStorer: Paths and dest lengths must be equal: %d != %d", len(paths), len(dest)))
	}
	errs = make([]error, len(paths))
	defer common.CatchException(&errs[0])

	q := queue.New(len(paths))
	for ix := range paths {
		p := fsutil.NormalizePath(paths[ix])
		q.Push(
			indexeddb.TransactionReadOnly,
			[]string{idbFileInfoStore},
			indexeddb.GetOp(idbFileInfoStore, i.jsPaths.Value(p)))
	}

	log.Debug("Loading file infos from JS: ", paths)
	infos, err := q.Do(i.db)
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

func (i *indexedDBStorer) QueueSetFileRecord(path string, data *storer.FileRecord) <-chan error {
	path = fsutil.NormalizePath(path)
	isRoot := path == "." || path == afero.FilePathSeparator
	if data == nil && isRoot {
		// cannot delete root dir
		err := make(chan error, 1)
		err <- syscall.ENOSYS
		close(err)
		return err
	}
	return i.queueSetFile(i.setQueue, path, data)
}

func (i *indexedDBStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	path = fsutil.NormalizePath(path)
	isRoot := path == "." || path == afero.FilePathSeparator
	if data == nil && isRoot {
		return syscall.ENOSYS // cannot delete root dir
	}
	return i.setFile(path, data)
}

func (i *indexedDBStorer) setFile(path string, data *storer.FileRecord) error {
	const maxQueue = 8 // arbitrarily large for a single file. only expect 2-3 operations
	q := queue.New(maxQueue)
	_ = i.queueSetFile(q, path, data)
	_, err := q.Do(i.db)
	if err != nil {
		// TODO Verify if AbortError type. If it isn't, then don't replace with syscall.ENOTDIR.
		// Should be the only reason for an abort. Later use an error handling mechanism in indexeddb pkg.
		log.Error("Aborted set file: ", err)
		err = syscall.ENOTDIR
	}
	return err
}

func (i *indexedDBStorer) queueSetFile(q *queue.Queue, path string, data *storer.FileRecord) <-chan error {
	if data == nil {
		q.Push(indexeddb.TransactionReadWrite, []string{idbFileInfoStore}, indexeddb.DeleteOp(idbFileInfoStore, i.jsPaths.Value(path)))
		_, err := q.Push(indexeddb.TransactionReadWrite, []string{idbFileContentsStore}, indexeddb.DeleteOp(idbFileContentsStore, i.jsPaths.Value(path)))
		return err
	}

	if !data.Mode.IsDir() {
		// this is a file, so include file contents
		q.Push(indexeddb.TransactionReadWrite, []string{idbFileContentsStore}, indexeddb.PutOp(
			idbFileContentsStore,
			i.jsPaths.Value(path), data.Data().JSValue(),
		))
	}
	fileInfo := map[string]interface{}{
		"ModTime": data.ModTime.Unix(),
		"Mode":    uint32(data.Mode),
		"Size":    data.Size(),
	}
	if path != afero.FilePathSeparator {
		fileInfo[idbParentKey] = filepath.Dir(path)
	}
	// include metadata update
	_, err := q.Push(indexeddb.TransactionReadWrite, []string{idbFileInfoStore}, indexeddb.PutOp(
		idbFileInfoStore,
		i.jsPaths.Value(path),
		js.ValueOf(fileInfo),
	))

	// verify a parent directory exists (except for root dir)
	dir := filepath.Dir(path)
	if dir != "" && dir != afero.FilePathSeparator {
		_, err = q.Push(indexeddb.TransactionReadOnly, []string{idbFileInfoStore}, i.batchRequireDir(dir))
	}
	return err
}

func (i *indexedDBStorer) batchRequireDir(path string) func(*indexeddb.Transaction) *indexeddb.Request {
	batchGet := indexeddb.GetOp(idbFileInfoStore, i.jsPaths.Value(path))
	return func(txn *indexeddb.Transaction) *indexeddb.Request {
		req := batchGet(txn)
		req.ListenSuccess(func() {
			result := req.Result()
			mode := i.getMode(result)
			if !mode.IsDir() {
				err := txn.Abort()
				if err != nil {
					panic(err)
				}
			}
		})
		return req
	}
}
