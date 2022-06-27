// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) chmod(args []js.Value) ([]interface{}, error) {
	_, err := s.chmodSync(args)
	return nil, err
}

func (s fileShim) chmodSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}

	path := args[0].String()
	mode := os.FileMode(args[1].Int())
	return nil, s.process.Files().Chmod(path, mode)
}
