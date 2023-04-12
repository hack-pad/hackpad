//go:build js
// +build js

package fs

import (
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func utimes(args []js.Value) ([]interface{}, error) {
	_, err := utimesSync(args)
	return nil, err
}

func utimesSync(args []js.Value) (interface{}, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Invalid number of args, expected 3: %v", args)
	}

	path := args[0].String()
	atime := time.Unix(int64(args[1].Int()), 0)
	mtime := time.Unix(int64(args[2].Int()), 0)
	p := process.Current()
	return nil, p.Files().Utimes(path, atime, mtime)
}
