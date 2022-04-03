// +build js

package process

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func (s processShim) cwd(args []js.Value) (interface{}, error) {
	return s.process.WorkingDirectory(), nil
}

func (s processShim) chdir(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("a new directory argument is required")
	}
	return nil, s.process.SetWorkingDirectory(args[0].String())
}
