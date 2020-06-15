package process

import (
	"syscall/js"

	"github.com/pkg/errors"
)

var (
	jsWasm     = js.Global().Get("WebAssembly")
	jsGo       = js.Global().Get("Go")
	uint8Array = js.Global().Get("Uint8Array")
)

func spawn(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	command := args[0].String()
	var argv []string
	length := args[1].Length()
	for i := 0; i < length; i++ {
		argv = append(argv, args[1].Index(i).String())
	}
	return Spawn(command, argv)
}

func Spawn(command string, args []string) (*Process, error) {
	process := New(command, args)
	return process, process.Start()
}
