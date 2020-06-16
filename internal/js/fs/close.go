package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/process"
	"github.com/pkg/errors"
)

func closeFn(args []js.Value) ([]interface{}, error) {
	ret, err := closeSync(args)
	return []interface{}{ret}, err
}

func closeSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("not enough args %d", len(args))
	}

	fd := fs.FID(args[0].Int())
	p := process.Current()
	err := p.Files().Close(fd)
	return nil, err
}
