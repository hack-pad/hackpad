package process

import (
	"os"
	"os/exec"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

func (p *process) startWasm() error {
	pids[p.pid] = p
	log.Printf("Spawning process: %v", p)
	command, err := exec.LookPath(p.command)
	if err != nil {
		return err
	}
	buf, err := p.Files().ReadFile(command)
	if err != nil {
		return err
	}
	go p.runWasmBytes(buf)
	return nil
}

func (p *process) Done() {
	log.Error("PID ", p.pid, " is done.\n", p.fileDescriptors)
	p.fileDescriptors.CloseAll()
	close(p.done)
}

func (p *process) runWasmBytes(wasm []byte) {
	handleErr := func(err error) {
		p.state = stateDone
		if err != nil {
			log.Errorf("Failed to start process: %s", err.Error())
			p.err = err
			p.state = "error"
		}
		p.Done()
	}

	p.state = stateCompiling
	goInstance := jsGo.New()
	goInstance.Set("argv", interop.SliceFromStrings(p.args))
	if p.attr.Env == nil {
		p.attr.Env = splitEnvPairs(os.Environ())
	}
	goInstance.Set("env", interop.StringMap(p.attr.Env))

	importObject := goInstance.Get("importObject")
	jsBuf := uint8Array.New(len(wasm))
	js.CopyBytesToJS(jsBuf, wasm)
	// TODO add module caching
	instantiatePromise := promise.New(jsWasm.Call("instantiate", jsBuf, importObject))
	module, err := promise.Await(instantiatePromise)
	if err != nil {
		handleErr(err)
		return
	}

	exports := module.Get("instance").Get("exports")

	wrapperExports := map[string]interface{}{
		"run": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer interop.PanicLogger()
			prev := switchContext(p.pid)
			ret := exports.Call("run", interop.SliceFromJSValues(args)...)
			switchContext(prev)
			return ret
		}),
		"resume": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			defer interop.PanicLogger()
			prev := switchContext(p.pid)
			ret := exports.Call("resume", interop.SliceFromJSValues(args)...)
			switchContext(prev)
			return ret
		}),
	}
	for export, value := range interop.Entries(exports) {
		_, overridden := wrapperExports[export]
		if !overridden {
			wrapperExports[export] = value
		}
	}
	instance := js.ValueOf(map[string]interface{}{ // Instance.exports is read-only, so create a shim
		"exports": wrapperExports,
	})

	p.state = stateRunning
	runPromise := promise.New(goInstance.Call("run", instance))
	_, err = promise.Await(runPromise)
	handleErr(err)
}
