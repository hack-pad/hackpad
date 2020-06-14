package process

import (
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var (
	lastPID    = atomic.NewUint64(minPID)
	jsWasm     = js.Global().Get("WebAssembly")
	jsGo       = js.Global().Get("Go").New()
	uint8Array = js.Global().Get("Uint8Array")
)

func spawn(args []js.Value) (interface{}, error) {
	return spawnWait(args, false)
}

func spawnSync(args []js.Value) (interface{}, error) {
	return spawnWait(args, true)
}

func spawnWait(args []js.Value, wait bool) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	command := args[0].String()
	var argv []string
	length := args[1].Length()
	for i := 0; i < length; i++ {
		argv = append(argv, args[1].Index(i).String())
	}
	pid, err := Spawn(command, argv, wait)
	return map[string]interface{}{
		"pid": pid,
	}, err
}

func Spawn(command string, args []string, wait bool) (pid uint64, err error) {
	pid = lastPID.Inc()
	log.Print("Spawning process: ", command, " ", strings.Join(args, " "))
	err = runWasm(command, args, wait)
	return pid, err
}

func runWasm(path string, args []string, wait bool) error {
	buf, err := fs.ReadFile(path)
	if err != nil {
		return err
	}
	importObject := jsGo.Get("importObject")
	jsBuf := uint8Array.New(len(buf))
	js.CopyBytesToJS(jsBuf, buf)
	// TODO add module caching
	instantiatePromise := jsWasm.Call("instantiate", jsBuf, importObject)
	fn := func() error {
		module, err := interop.Await(instantiatePromise)
		if err != nil {
			return err
		}
		runPromise := jsGo.Call("run", module)
		_, err = interop.Await(runPromise)
		return err
	}
	if !wait {
		go fn()
		return nil
	}
	return fn()
}
