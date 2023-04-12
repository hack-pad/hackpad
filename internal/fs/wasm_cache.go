//go:build js
// +build js

package fs

import (
	"io"
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpad/internal/fsutil"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

var jsWasm = js.Global().Get("WebAssembly")

type wasmCacheFs struct {
	rootFs
	memCache map[string]js.Value
}

func init() {
	initWasmCache()
}

func initWasmCache() {
	fs, err := newWasmCacheFs(filesystem)
	if err != nil {
		log.Error("Failed to enable Wasm Module cache: ", err)
	} else {
		filesystem = fs
	}
}

func shouldCache(path string) bool {
	return strings.HasPrefix(path, "usr/local/go/")
}

func newWasmCacheFs(underlying rootFs) (*wasmCacheFs, error) {
	return &wasmCacheFs{
		rootFs:   underlying,
		memCache: make(map[string]js.Value),
	}, nil
}

func (w *wasmCacheFs) readFile(path string) (blob.Blob, error) {
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
	log.Debug("Checking wasm instance cache")
	module, memCacheHit := w.memCache[path]
	if memCacheHit {
		log.Debug("memCache hit: ", path)
	} else {
		log.Debug("memCache miss: ", path)
		moduleBlob, err := w.readFile(path)
		if err != nil {
			log.Debug("reading file failed: ", path)
			return js.Value{}, err
		}
		module = idbblob.FromBlob(moduleBlob).JSValue()
		if !module.Truthy() {
			log.Debug("fs miss: ", path, module.Length())
		}
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

	if shouldCache(path) {
		w.memCache[path] = result.Get("module") // save compiled module for reuse
	}
	return result.Get("instance"), nil
}

func (w *wasmCacheFs) dropModuleCache(path string) error {
	path = fsutil.NormalizePath(path)
	delete(w.memCache, path)
	return nil
}

func (w *wasmCacheFs) Create(name string) (hackpadfs.File, error) {
	if err := w.dropModuleCache(fsutil.NormalizePath(name)); err != nil {
		return nil, err
	}
	return hackpadfs.Create(w.rootFs, name)
}

func (w *wasmCacheFs) OpenFile(name string, flag int, perm os.FileMode) (hackpadfs.File, error) {
	if flag != os.O_RDONLY {
		err := w.dropModuleCache(fsutil.NormalizePath(name))
		if err != nil {
			return nil, err
		}
	}
	return hackpadfs.OpenFile(w.rootFs, name, flag, perm)
}

func (w *wasmCacheFs) Remove(name string) error {
	if err := w.dropModuleCache(fsutil.NormalizePath(name)); err != nil {
		return err
	}
	return hackpadfs.Remove(w.rootFs, name)
}

func (w *wasmCacheFs) RemoveAll(path string) error {
	// TODO is there a performant way to remove modules recursively?
	if err := w.dropModuleCache(fsutil.NormalizePath(path)); err != nil {
		return err
	}
	return hackpadfs.RemoveAll(w.rootFs, path)
}

func (w *wasmCacheFs) Rename(oldname, newname string) error {
	// TODO maybe preserve oldname's module somehow?
	if err := w.dropModuleCache(fsutil.NormalizePath(oldname)); err != nil {
		return err
	}
	if err := w.dropModuleCache(fsutil.NormalizePath(newname)); err != nil {
		return err
	}
	return hackpadfs.Rename(w.rootFs, oldname, newname)
}

func (w *wasmCacheFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return hackpadfs.Chtimes(w.rootFs, name, atime, mtime)
}
