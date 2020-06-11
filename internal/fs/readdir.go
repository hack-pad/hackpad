package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func readdir(args []js.Value) ([]interface{}, error) {
	fileNames, err := readdirSync(args)
	return []interface{}{fileNames}, err
}

func readdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	dir, err := ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []interface{}
	for _, f := range dir {
		names = append(names, f.Name())
	}
	return names, err
}

func ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(filesystem, path)
}
