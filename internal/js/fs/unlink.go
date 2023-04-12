//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

// unlink(path, callback) { callback(enosys()); },

func unlink(args []js.Value) ([]interface{}, error) {
	_, err := unlinkSync(args)
	return nil, err
}

func unlinkSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	p := process.Current()
	return nil, p.Files().Unlink(path)
}
