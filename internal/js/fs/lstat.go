package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
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
	info, err := fs.Lstat(path)
	return jsStat(info), err
}
