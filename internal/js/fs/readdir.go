//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
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
	p := process.Current()
	dir, err := p.Files().ReadDir(path)
	if err != nil {
		return nil, err
	}
	var names []interface{}
	for _, f := range dir {
		names = append(names, f.Name())
	}
	return names, err
}
