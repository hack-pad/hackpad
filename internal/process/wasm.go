// +build js

package process

import (
	"os"
	"strings"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

var wasmCache = make(map[string]wasmCacheValue)

var (
	jsObject = js.Global().Get("Object")
)

type wasmCacheValue struct {
	modTime time.Time
	module  js.Value
}

func cacheAllowed(path string) bool {
	return strings.HasPrefix(path, "/go/")
}

func (p *process) loadWasmModule(path string) (js.Value, error) {
	allowCache := cacheAllowed(path)
	var info os.FileInfo
	if allowCache {
		var err error
		info, err = p.Files().Stat(path)
		if err != nil {
			return js.Value{}, err
		}
		val, ok := wasmCache[path]
		if ok && info.ModTime() == val.modTime {
			return val.module, nil
		}
	}

	wasm, err := p.Files().ReadFile(path)
	if err != nil {
		return js.Value{}, err
	}
	jsBuf := interop.NewByteArray(wasm)
	compilePromise := promise.From(jsWasm.Call("compile", jsBuf))
	module, err := compilePromise.Await()
	if err != nil {
		return js.Value{}, err
	}
	jsModule := module.(js.Value)

	if allowCache {
		wasmCache[path] = wasmCacheValue{
			modTime: info.ModTime(),
			module:  jsModule,
		}
	}
	return jsModule, nil
}

func (p *process) run(path string) {
	exitChan := make(chan int, 1)
	runPromise, err := p.startWasmPromise(path, exitChan)
	if err != nil {
		p.handleErr(err)
		return
	}
	_, err = runPromise.Await()
	p.exitCode = <-exitChan
	p.handleErr(err)
}

func (p *process) startWasmPromise(path string, exitChan chan<- int) (promise.Promise, error) {
	module, err := p.loadWasmModule(path)
	if err != nil {
		return nil, err
	}

	p.state = stateCompiling
	goInstance := jsGo.New()
	goInstance.Set("argv", interop.SliceFromStrings(p.args))
	if p.attr.Env == nil {
		p.attr.Env = splitEnvPairs(os.Environ())
	}
	goInstance.Set("env", interop.StringMap(p.attr.Env))
	var resumeFuncPtr *js.Func
	goInstance.Set("exit", interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
		defer func() {
			if resumeFuncPtr != nil {
				resumeFuncPtr.Release()
			}
		}()
		if len(args) == 0 {
			exitChan <- -1
			return nil
		}
		code := args[0].Int()
		exitChan <- code
		if code != 0 {
			log.Warnf("Process exited with code %d: %s", code, p)
		}
		return nil
	}))

	importObject := goInstance.Get("importObject")
	time.Sleep(1) // nolint:staticcheck // allow JS event loop to run
	instantiatePromise := promise.From(jsWasm.Call("instantiate", module, importObject))
	instanceInterface, err := instantiatePromise.Await()
	if err != nil {
		return nil, err
	}
	instance := instanceInterface.(js.Value)

	exports := instance.Get("exports")

	resumeFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer interop.PanicLogger()
		prev := switchContext(p.pid)
		ret := exports.Call("resume", interop.SliceFromJSValues(args)...)
		switchContext(prev)
		return ret
	})
	resumeFuncPtr = &resumeFunc
	wrapperExports := map[string]interface{}{
		"run": interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
			defer interop.PanicLogger()
			prev := switchContext(p.pid)
			ret := exports.Call("run", interop.SliceFromJSValues(args)...)
			switchContext(prev)
			return ret
		}),
		"resume": resumeFunc,
	}
	for export, value := range interop.Entries(exports) {
		_, overridden := wrapperExports[export]
		if !overridden {
			wrapperExports[export] = value
		}
	}
	wrapperInstance := jsObject.Call("defineProperty",
		jsObject.Call("create", instance),
		"exports", map[string]interface{}{ // Instance.exports is read-only, so create a shim
			"value":    wrapperExports,
			"writable": false,
		},
	)

	time.Sleep(1) // nolint:staticcheck // allow JS event loop to run
	p.state = stateRunning
	return promise.From(goInstance.Call("run", wrapperInstance)), nil
}
