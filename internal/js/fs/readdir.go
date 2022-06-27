// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) readdir(args []js.Value) ([]interface{}, error) {
	fileNames, err := s.readdirSync(args)
	return []interface{}{fileNames}, err
}

func (s fileShim) readdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	path := args[0].String()
	dir, err := s.process.Files().ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []interface{}
	for _, f := range dir {
		names = append(names, f.Name())
	}
	return names, err
}
