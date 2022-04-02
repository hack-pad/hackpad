// +build js

package process

import (
	"os"
	"runtime"
	"strings"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
)

var (
	jsObject = js.Global().Get("Object")
)

func (p *Process) newWasmInstance(path string, importObject js.Value) (js.Value, error) {
	return p.Files().WasmInstance(path, importObject)
}

func (p *Process) run(path string) {
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

func (p *Process) startWasmPromise(path string, exitChan chan<- int) (promise.Promise, error) {
	p.state = stateCompiling
	goInstance := jsGo.New()
	goInstance.Set("argv", interop.SliceFromStrings(p.args))
	if p.env == nil {
		p.env = splitEnvPairs(os.Environ())
	}
	goInstance.Set("env", interop.StringMap(p.env))
	var resumeFuncPtr *js.Func
	goInstance.Set("exit", interop.SingleUseFunc(func(this js.Value, args []js.Value) interface{} {
		defer func() {
			if resumeFuncPtr != nil {
				resumeFuncPtr.Release()
			}
			// TODO exit hook for worker

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

	p.state = stateRunning
	return promise.From(goInstance.Call("run", instance)), nil
}

func splitEnvPairs(pairs []string) map[string]string {
	env := make(map[string]string)
	for _, pair := range pairs {
		equalIndex := strings.IndexRune(pair, '=')
		if equalIndex == -1 {
			env[pair] = ""
		} else {
			key, value := pair[:equalIndex], pair[equalIndex+1:]
			env[key] = value
		}
	}
	return env
}
