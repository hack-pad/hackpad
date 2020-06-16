package process

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/process"
	"github.com/pkg/errors"
)

func cwd(args []js.Value) (interface{}, error) {
	return process.Current().WorkingDirectory(), nil
}

func chdir(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("a new directory argument is required")
	}
	newCWD := args[0].String()
	p := process.Current()
	info, err := p.Files().Stat(newCWD)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.Errorf("%s is not a directory", info.Name())
	}
	return nil, p.SetWorkingDirectory(args[0].String())
}
