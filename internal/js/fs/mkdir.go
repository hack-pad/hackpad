// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) mkdir(args []js.Value) ([]interface{}, error) {
	_, err := s.mkdirSync(args)
	return nil, err
}

func (s fileShim) mkdirSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	path := args[0].String()
	options := args[1]
	var mode os.FileMode
	switch {
	case options.Type() == js.TypeNumber:
		mode = os.FileMode(options.Int())
	case options.Type() == js.TypeObject && options.Get("mode").Truthy():
		mode = os.FileMode(options.Get("mode").Int())
	default:
		mode = 0777
	}
	recursive := false
	if options.Type() == js.TypeObject && options.Get("recursive").Truthy() {
		recursive = true
	}

	if recursive {
		return nil, s.process.Files().MkdirAll(path, mode)
	}
	return nil, s.process.Files().Mkdir(path, mode)
}
