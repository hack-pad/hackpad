// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func (s fileShim) open(args []js.Value) ([]interface{}, error) {
	fd, err := s.openSync(args)
	return []interface{}{fd}, err
}

func (s fileShim) openSync(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.Errorf("Expected path, received: %v", args)
	}
	path := args[0].String()
	flags := os.O_RDONLY
	if len(args) >= 2 {
		flags = args[1].Int()
	}
	mode := os.FileMode(0666)
	if len(args) >= 3 {
		mode = os.FileMode(args[2].Int())
	}

	fd, err := s.process.Files().Open(path, flags, mode)
	return fd, err
}
