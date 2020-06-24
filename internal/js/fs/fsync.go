package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/process"
	"github.com/pkg/errors"
)

// fsync(fd, callback) { callback(null); },

func fsync(args []js.Value) ([]interface{}, error) {
	_, err := fsyncSync(args)
	return nil, err
}

func fsyncSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := fs.FID(args[0].Int())
	p := process.Current()
	return nil, p.Files().Fsync(fd)
}
