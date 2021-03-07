// +build js

package fs

import (
	"io"
	"os"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/indexeddb"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
	"github.com/spf13/afero"
)

var jsWasm = js.Global().Get("WebAssembly")

const (
	wasmCacheDB      = "wasmModules"
	wasmCacheVersion = 1
	wasmCacheStore   = "modules"
)

type wasmCacheFs struct {
	rootFs
	db       *indexeddb.DB
	jsPaths  interop.StringCache
	memCache map[string]js.Value
	idbFound map[string]bool
}

func init() {
	go initWasmCache()
}

func initWasmCache() {
	fs, err := newWasmCacheFs(filesystem)
	if err != nil {
		log.Error("Failed to enable Wasm Module cache: ", err)
	} else {
		filesystem = fs
	}
}

func newWasmCacheFs(underlying rootFs) (*wasmCacheFs, error) {
	db, err := indexeddb.New(wasmCacheDB, wasmCacheVersion, func(db *indexeddb.DB, oldVersion, newVersion int) error {
		_, err := db.CreateObjectStore(wasmCacheStore, indexeddb.ObjectStoreOptions{})
		return err
	})
	return &wasmCacheFs{
		rootFs:   underlying,
		db:       db,
		memCache: make(map[string]js.Value),
		idbFound: make(map[string]bool),
	}, err
}

func (w *wasmCacheFs) readFile(path string) (blob.Blob, error) {
	path = fsutil.NormalizePath(path)
	f, err := w.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	buf, _, err := blob.Read(f, int(info.Size()))
	if err == io.EOF {
		err = nil
	}
	return buf, err
}

func (w *wasmCacheFs) WasmInstance(path string, importObject js.Value) (js.Value, error) {
	path = fsutil.NormalizePath(path)
	log.Debug("Checking wasm instance cache")
	module, memCacheHit := w.memCache[path]
	if memCacheHit {
		log.Debug("memCache hit: ", path)
	} else {
		log.Debug("memCache miss: ", path)
		dbFound, dbChecked := w.idbFound[path]
		if dbFound || !dbChecked {
			moduleBlob, err := w.getModuleBytes(path)
			if err != nil {
				log.Debug("Failed to look up module at path: ", path)
				return js.Value{}, err
			}
			if moduleBlob != nil {
				module = moduleBlob.JSValue()
			}
		}
	}

	dbCacheHit := module.Truthy()
	if dbCacheHit {
		log.Debug("dbCache hit: ", path, module.Length())
	} else {
		log.Debug("dbCache miss: ", path)
		moduleBlob, err := w.readFile(path)
		if err != nil {
			log.Debug("reading file failed: ", path)
			return js.Value{}, err
		}
		module = moduleBlob.JSValue()
	}

	instantiatePromise := promise.From(jsWasm.Call("instantiate", module, importObject))
	instanceInterface, err := instantiatePromise.Await()
	if err != nil {
		return js.Value{}, err
	}
	result := instanceInterface.(js.Value)

	log.Debug("successfully instantiated module: ", path)
	if memCacheHit {
		// if memCacheHit, then module is already compiled
		// so return value is only an Instance, not a ResultObject
		return result, nil
	}

	w.memCache[path] = result.Get("module") // save compiled module for reuse
	instance := result.Get("instance")
	if !dbCacheHit {
		// save module to IDB
		return instance, w.setModule(path, module)
	}
	return instance, nil
}

func (w *wasmCacheFs) dropModuleCache(path string) error {
	path = fsutil.NormalizePath(path)

	_, memCacheHit := w.memCache[path]
	if memCacheHit {
		delete(w.memCache, path)
	}

	dbCacheHit := w.idbFound[path]
	if dbCacheHit {
		err := w.deleteModule(path)
		if err != nil {
			return err
		}
		delete(w.idbFound, path)
	}
	return nil
}

func (w *wasmCacheFs) deleteModule(path string) error {
	txn, err := w.db.Transaction(indexeddb.TransactionReadWrite, wasmCacheStore)
	if err != nil {
		return err
	}
	store, err := txn.ObjectStore(wasmCacheStore)
	if err != nil {
		return err
	}
	req, err := store.Delete(w.jsPaths.Value(path))
	if err != nil {
		return err
	}
	_, err = req.Await()
	return err
}

func (w *wasmCacheFs) setModule(path string, module js.Value) error {
	path = fsutil.NormalizePath(path)
	txn, err := w.db.Transaction(indexeddb.TransactionReadWrite, wasmCacheStore)
	if err != nil {
		return err
	}
	store, err := txn.ObjectStore(wasmCacheStore)
	if err != nil {
		return err
	}
	req, err := store.Put(w.jsPaths.Value(path), module.JSValue())
	if err != nil {
		return err
	}
	_, err = req.Await()
	if err == nil {
		w.idbFound[path] = true
	}
	return err
}

func (w *wasmCacheFs) getModuleBytes(path string) (blob.Blob, error) {
	path = fsutil.NormalizePath(path)
	txn, err := w.db.Transaction(indexeddb.TransactionReadOnly, wasmCacheStore)
	if err != nil {
		return nil, err
	}
	store, err := txn.ObjectStore(wasmCacheStore)
	if err != nil {
		return nil, err
	}
	req, err := store.Get(w.jsPaths.Value(path))
	if err != nil {
		return nil, err
	}
	jsBytes, err := req.Await()
	if err != nil || jsBytes.IsUndefined() {
		return nil, err
	}
	w.idbFound[path] = true
	return blob.NewFromJS(jsBytes)
}

func (w *wasmCacheFs) Create(name string) (afero.File, error) {
	if err := w.dropModuleCache(fsutil.NormalizePath(name)); err != nil {
		return nil, err
	}
	return w.rootFs.Create(name)
}

func (w *wasmCacheFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if flag != os.O_RDONLY {
		err := w.dropModuleCache(fsutil.NormalizePath(name))
		if err != nil {
			return nil, err
		}
	}
	return w.rootFs.OpenFile(name, flag, perm)
}

func (w *wasmCacheFs) Remove(name string) error {
	if err := w.dropModuleCache(fsutil.NormalizePath(name)); err != nil {
		return err
	}
	return w.rootFs.Remove(name)
}

func (w *wasmCacheFs) RemoveAll(path string) error {
	// TODO is there a performant way to remove modules recursively?
	if err := w.dropModuleCache(fsutil.NormalizePath(path)); err != nil {
		return err
	}
	return w.rootFs.RemoveAll(path)
}

func (w *wasmCacheFs) Rename(oldname, newname string) error {
	// TODO maybe preserve oldname's module somehow?
	if err := w.dropModuleCache(fsutil.NormalizePath(oldname)); err != nil {
		return err
	}
	if err := w.dropModuleCache(fsutil.NormalizePath(newname)); err != nil {
		return err
	}
	return w.rootFs.Rename(oldname, newname)
}

func (w *wasmCacheFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return w.rootFs.Chtimes(name, atime, mtime)
}

func (w *wasmCacheFs) DestroyMount(path string) error {
	txn, err := w.db.Transaction(indexeddb.TransactionReadWrite, wasmCacheStore)
	if err != nil {
		return err
	}
	store, err := txn.ObjectStore(wasmCacheStore)
	if err != nil {
		return err
	}
	req, err := store.Clear()
	if err != nil {
		return err
	}
	_, err = req.Await()
	if err != nil {
		return err
	}
	w.idbFound = make(map[string]bool)
	w.memCache = make(map[string]js.Value)
	return w.rootFs.DestroyMount(path)
}
