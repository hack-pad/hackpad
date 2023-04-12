//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func rmdir(args []js.Value) ([]interface{}, error) {
	_, err := rmdirSync(args)
	return nil, err
}

func rmdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	p := process.Current()
	return nil, p.Files().RemoveDir(path)
}
