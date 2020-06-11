package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
)

func fstat(args []js.Value) ([]interface{}, error) {
	info, err := fstatSync(args)
	return []interface{}{info}, err
}

func fstatSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	fd := uint64(args[0].Int())
	info, err := Fstat(fd)
	return jsStat(info), err
}

func Fstat(fd uint64) (os.FileInfo, error) {
	fileDescriptor := fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return nil, errors.Errorf("unknown fd: %d", fd)
	}
	return fileDescriptor.file.Stat()
}
