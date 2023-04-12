//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func chown(args []js.Value) ([]interface{}, error) {
	_, err := chownSync(args)
	return nil, err
}

func chownSync(args []js.Value) (interface{}, error) {
	if len(args) != 3 {
		return nil, errors.Errorf("Invalid number of args, expected 3: %v", args)
	}

	path := args[0].String()
	uid := args[1].Int()
	gid := args[2].Int()
	return nil, Chown(path, uid, gid)
}

func Chown(path string, uid, gid int) error {
	// TODO no-op, consider adding user and group ID support to hackpadfs
	return nil
}
