package fs

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func flock(args []js.Value) ([]interface{}, error) {
	_, err := flockSync(args)
	return nil, err
}

func flockSync(args []js.Value) (interface{}, error) {
	if len(args) != 2 {
		return nil, errors.Errorf("Invalid number of args, expected 2: %v", args)
	}
	fd := args[0].Int()
	flags := args[1].Int()

	return nil, Flock(fd, flags)
}

func Flock(fd, flags int) error {
	// TODO implement flock's
	return nil
}
