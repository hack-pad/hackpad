//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

// rename(from, to, callback) { callback(enosys()); },

func rename(args []js.Value) ([]interface{}, error) {
	_, err := renameSync(args)
	return nil, err
}

func renameSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	oldPath := args[0].String()
	newPath := args[1].String()
	p := process.Current()
	return nil, p.Files().Rename(oldPath, newPath)
}
