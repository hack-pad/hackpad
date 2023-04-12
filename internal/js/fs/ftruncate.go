//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func ftruncateSync(args []js.Value) (interface{}, error) {
	_, err := ftruncate(args)
	return nil, err
}

func ftruncate(args []js.Value) ([]interface{}, error) {
	// args: fd, len
	if len(args) == 0 {
		return nil, errors.Errorf("missing required args, expected fd: %+v", args)
	}
	fd := fs.FID(args[0].Int())
	length := 0
	if len(args) >= 2 {
		length = args[1].Int()
	}

	p := process.Current()
	return nil, p.Files().Truncate(fd, int64(length))
}
