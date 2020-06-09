package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func mkdir(args []js.Value) ([]interface{}, error) {
	_, err := mkdirSync(args)
	return nil, err
}

func mkdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	path := args[0].String()
	mode := os.FileMode(args[1].Int())
	return nil, Mkdir(path, mode)
}

func Mkdir(path string, mode os.FileMode) error {
	// TODO find better mkdir implementation, memfs only supports all
	return filesystem.MkdirAll(path, mode)
}
