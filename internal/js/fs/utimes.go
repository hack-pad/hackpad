// +build js

package fs

import (
	"syscall/js"
	"time"

	"github.com/pkg/errors"
)

func (s fileShim) utimes(args []js.Value) ([]interface{}, error) {
	_, err := s.utimesSync(args)
	return nil, err
}

func (s fileShim) utimesSync(args []js.Value) (interface{}, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Invalid number of args, expected 3: %v", args)
	}

	path := args[0].String()
	atime := time.Unix(int64(args[1].Int()), 0)
	mtime := time.Unix(int64(args[2].Int()), 0)
	return nil, s.process.Files().Utimes(path, atime, mtime)
}
