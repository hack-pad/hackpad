//go:build js
// +build js

package process

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func cwd(args []js.Value) (interface{}, error) {
	return process.Current().WorkingDirectory(), nil
}

func chdir(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("a new directory argument is required")
	}
	p := process.Current()
	return nil, p.SetWorkingDirectory(args[0].String())
}
