// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

// unlink(path, callback) { callback(enosys()); },

func (s fileShim) unlink(args []js.Value) ([]interface{}, error) {
	_, err := s.unlinkSync(args)
	return nil, err
}

func (s fileShim) unlinkSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	return nil, s.process.Files().Unlink(path)
}
