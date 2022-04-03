// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/pkg/errors"
)

func (s fileShim) fstat(args []js.Value) ([]interface{}, error) {
	info, err := s.fstatSync(args)
	return []interface{}{info}, err
}

func (s fileShim) fstatSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := fs.FID(args[0].Int())
	info, err := s.process.Files().Fstat(fd)
	return jsStat(info), err
}
