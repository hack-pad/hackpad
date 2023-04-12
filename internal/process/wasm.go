//go:build js
// +build js

package process

import (
	"os"
	"runtime"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
)

var (
	jsObject = js.Global().Get("Object")
)

func (p *process) newWasmInstance(path string, importObject js.Value) (js.Value, error) {
	return p.Files().WasmInstance(path, importObject)
}

func (p *process) run(path string) {
	defer func() {
		go runtime.GC()
	}()

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
			// TODO free the whole goInstance to fix garbage issues entirely. Freeing individual properties appears to work for now, but is ultimately a bad long-term solution because memory still accumulates.
			goInstance.Set("mem", js.Null())
			goInstance.Set("importObject", js.Null())
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

	instance, err := p.newWasmInstance(path, importObject)
	if err != nil {
		return nil, err
	}

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

	p.state = stateRunning
	return promise.From(goInstance.Call("run", wrapperInstance)), nil
}
