// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/pkg/errors"
)

// fsync(fd, callback) { callback(null); },

func (s fileShim) fsync(args []js.Value) ([]interface{}, error) {
	_, err := s.fsyncSync(args)
	return nil, err
}

func (s fileShim) fsyncSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := fs.FID(args[0].Int())
	return nil, s.process.Files().Fsync(fd)
}
