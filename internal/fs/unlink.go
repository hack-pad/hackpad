package fs

import (
	"os"
	"syscall/js"

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
	return nil, Unlink(path)
}

func Unlink(path string) error {
	path = resolvePath(path)
	info, err := Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.ErrPermission
	}
	return filesystem.Remove(path)
}
