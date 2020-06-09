package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func stat(args []js.Value) ([]interface{}, error) {
	info, err := statSync(args)
	return []interface{}{info}, err
}

func statSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	info, err := Stat(path)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"mode": info.Mode(),
		"uid":  0, // TODO use real values for uid and gid
		"gid":  0,
		"size": info.Size(),
	}, nil
}

func Stat(path string) (os.FileInfo, error) {
	return filesystem.Stat(path)
}
