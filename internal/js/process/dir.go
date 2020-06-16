package process

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/js/fs"
	"github.com/pkg/errors"
)

func cwd(args []js.Value) (interface{}, error) {
	return interop.WorkingDirectory(), nil
}

func chdir(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("a new directory argument is required")
	}
	newCWD := args[0].String()
	info, err := fs.Stat(newCWD)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.Errorf("%s is not a directory", info.Name())
	}
	interop.SetWorkingDirectory(args[0].String())
	return nil, nil
}
