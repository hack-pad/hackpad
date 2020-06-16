package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
)

var ErrNotDir = interop.NewError("not a directory", "ENOTDIR")

func rmdir(args []js.Value) ([]interface{}, error) {
	_, err := rmdirSync(args)
	return nil, err
}

func rmdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	return nil, RemoveDir(path)
}

func RemoveDir(path string) error {
	path = resolvePath(path)
	info, err := Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}
	return filesystem.Remove(path)
}
