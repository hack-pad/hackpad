package fs

import (
	"os"
	"syscall/js"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

func open(args []js.Value) ([]interface{}, error) {
	fd, err := openSync(args)
	return []interface{}{fd}, err
}

func openSync(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.Errorf("Expected path, received: %v", args)
	}
	path := args[0].String()
	flags := os.O_RDONLY
	if len(args) >= 2 {
		flags = args[1].Int()
	}
	mode := os.FileMode(0666)
	if len(args) >= 3 {
		mode = os.FileMode(args[2].Int())
	}

	fd, err := Open(path, flags, mode)
	return fd, err
}

func Open(path string, flags int, mode os.FileMode) (fd uint64, err error) {
	path = resolvePath(path)
	file, err := getFile(path, flags, mode)
	if err != nil {
		return 0, err
	}
	return openWithFile(file)
}

func openWithFile(file afero.File) (uint64, error) {
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

func getFile(absPath string, flags int, mode os.FileMode) (afero.File, error) {
	switch absPath {
	case "/dev/null":
		return newNullFile(), nil
	}
	return filesystem.OpenFile(absPath, flags, mode)
}
