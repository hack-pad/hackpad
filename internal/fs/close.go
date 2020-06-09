package fs

import (
	"syscall/js"

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

	fd := uint64(args[0].Int())
	return nil, Close(fd)
}

func Close(fd uint64) error {
	fileDescriptor := fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return errors.Errorf("unknown fd: %d")
	}
	if fileDescriptor.openCount.Dec() == 0 {
		fileDescriptorMu.Lock()
		if fileDescriptor.openCount.Load() == 0 {
			delete(fileDescriptorIDs, fd)
			delete(fileDescriptorNames, fileDescriptor.file.Name())
		}
		fileDescriptorMu.Unlock()
		return fileDescriptor.file.Close()
	}
	return nil
}
