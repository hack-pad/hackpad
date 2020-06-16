package fs

import (
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/fs"
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
	return nil, fs.Utimes(path, atime, mtime)
}
