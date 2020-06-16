package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
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
	return nil, fs.RemoveDir(path)
}
