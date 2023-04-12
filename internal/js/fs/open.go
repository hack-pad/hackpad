//go:build js
// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func open(args []js.Value) ([]interface{}, error) {
	fd, err := openSync(args)
	return []interface{}{fd}, err
}

func openSync(args []js.Value) (interface{}, error) {
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

	p := process.Current()
	fd, err := p.Files().Open(path, flags, mode)
	return fd.JSValue(), err
}
