// +build js

package fs

import (
	"io"
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/fsutil"
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
	jsPaths  interop.StringCache
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
	return strings.HasPrefix(path, "/usr/local/go/")
}

func newWasmCacheFs(underlying rootFs) (*wasmCacheFs, error) {
	return &wasmCacheFs{
		rootFs:   underlying,
		memCache: make(map[string]js.Value),
	}, nil
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
		moduleBlob, err := w.readFile(path)
		if err != nil {
			log.Debug("reading file failed: ", path)
			return js.Value{}, err
		}
		module = moduleBlob.JSValue()
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
