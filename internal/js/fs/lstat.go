package fs

import (
	"os"
	"syscall/js"

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
	info, err := Lstat(path)
	return jsStat(info), err
}

func Lstat(path string) (os.FileInfo, error) {
	// TODO add proper symlink support
	return filesystem.Stat(resolvePath(path))
}
