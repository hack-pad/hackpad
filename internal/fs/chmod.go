package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func chmod(args []js.Value) ([]interface{}, error) {
	_, err := chmodSync(args)
	return nil, err
}

func chmodSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}

	path := args[0].String()
	mode := os.FileMode(args[1].Int())
	return nil, Chmod(path, mode)
}

func Chmod(path string, mode os.FileMode) error {
	return filesystem.Chmod(resolvePath(path), mode)
}
