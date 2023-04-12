//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func pipe(args []js.Value) ([]interface{}, error) {
	fds, err := pipeSync(args)
	return []interface{}{fds}, err
}

func pipeSync(args []js.Value) (interface{}, error) {
	if len(args) != 0 {
		return nil, errors.Errorf("Invalid number of args, expected 0: %v", args)
	}
	p := process.Current()
	fds := p.Files().Pipe()
	return []interface{}{fds[0].JSValue(), fds[1].JSValue()}, nil
}
