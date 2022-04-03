// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

// rename(from, to, callback) { callback(enosys()); },

func (s fileShim) rename(args []js.Value) ([]interface{}, error) {
	_, err := s.renameSync(args)
	return nil, err
}

func (s fileShim) renameSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	oldPath := args[0].String()
	newPath := args[1].String()
	return nil, s.process.Files().Rename(oldPath, newPath)
}
