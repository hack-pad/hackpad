// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) lstat(args []js.Value) ([]interface{}, error) {
	info, err := s.lstatSync(args)
	return []interface{}{info}, err
}

func (s fileShim) lstatSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	info, err := s.process.Files().Lstat(path)
	return jsStat(info), err
}
