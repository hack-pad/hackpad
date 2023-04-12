//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

// fsync(fd, callback) { callback(null); },

func fsync(args []js.Value) ([]interface{}, error) {
	_, err := fsyncSync(args)
	return nil, err
}

func fsyncSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := fs.FID(args[0].Int())
	p := process.Current()
	return nil, p.Files().Fsync(fd)
}
