//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func fstat(args []js.Value) ([]interface{}, error) {
	info, err := fstatSync(args)
	return []interface{}{info}, err
}

func fstatSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := fs.FID(args[0].Int())
	p := process.Current()
	info, err := p.Files().Fstat(fd)
	return jsStat(info), err
}
