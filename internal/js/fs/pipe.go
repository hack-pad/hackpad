package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/process"
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
	fds, err := p.Files().Pipe()
	if err != nil {
		return nil, err
	}
	return []interface{}{fds[0], fds[1]}, err
}
