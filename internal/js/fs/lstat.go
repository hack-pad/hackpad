//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func lstat(args []js.Value) ([]interface{}, error) {
	info, err := lstatSync(args)
	return []interface{}{info}, err
}

func lstatSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	p := process.Current()
	info, err := p.Files().Lstat(path)
	return jsStat(info), err
}
