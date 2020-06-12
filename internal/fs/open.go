package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

func open(args []js.Value) ([]interface{}, error) {
	fd, err := openSync(args)
	return []interface{}{fd}, err
}

func openSync(args []js.Value) (interface{}, error) {
	if len(args) < 3 {
		return nil, errors.Errorf("not enough args %d", len(args))
	}

	path := args[0].String()
	flags := args[1].Int()
	mode := args[2].Int()

	fd, err := Open(path, flags, mode)
	return fd, err
}

func Open(path string, flags, mode int) (fd uint64, err error) {
	path = resolvePath(path)
	file, err := filesystem.OpenFile(path, flags, os.FileMode(mode))
	if err != nil {
		return 0, err
	}
	if fileDescriptorNames[file.Name()] == nil {
		fileDescriptorMu.Lock()
		if fileDescriptorNames[file.Name()] == nil {
			fd := &fileDescriptor{
				id:        lastFileDescriptorID.Inc() - 1,
				file:      file,
				openCount: atomic.NewUint64(0),
			}
			fileDescriptorNames[file.Name()] = fd
			fileDescriptorIDs[fd.id] = fd
		}
		fileDescriptorMu.Unlock()
	}
	descriptor := fileDescriptorNames[file.Name()]
	descriptor.openCount.Inc()
	return descriptor.id, nil
}
