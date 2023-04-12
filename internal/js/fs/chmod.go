//go:build js
// +build js

package fs

import (
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
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
	p := process.Current()
	return nil, p.Files().Chmod(path, mode)
}
