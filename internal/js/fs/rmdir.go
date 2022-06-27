// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) rmdir(args []js.Value) ([]interface{}, error) {
	_, err := s.rmdirSync(args)
	return nil, err
}

func (s fileShim) rmdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	return nil, s.process.Files().RemoveDir(path)
}
