package fs

import (
	"io"
	"syscall/js"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func read(args []js.Value) ([]interface{}, error) {
	n, buf, err := readSyncImpl(args)
	return []interface{}{n, buf}, err
}

func readSync(args []js.Value) (interface{}, error) {
	n, _, err := readSyncImpl(args)
	return n, err
}

func readSyncImpl(args []js.Value) (int, js.Value, error) {
	// args: fd, buffer, offset, length, position
	if len(args) != 5 {
		return 0, js.Null(), errors.Errorf("missing required args, expected 5: %+v", args)
	}
	fd := uint64(args[0].Int())
	jsBuffer := args[1]
	offset := args[2].Int()
	length := args[3].Int()
	var position *int64
	if args[4].Type() == js.TypeNumber {
		position = new(int64)
		*position = int64(args[4].Int())
	}

	buffer := make([]byte, length)
	n, err := Read(fd, buffer, offset, length, position)
	js.CopyBytesToJS(jsBuffer, buffer)
	return n, jsBuffer, err
}

func Read(fd uint64, buffer []byte, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return 0, errors.Errorf("unknown fd: %d", fd)
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	if position == nil {
		n, err = fileDescriptor.file.Read(buffer[offset : offset+length])
	} else {
		n, err = fileDescriptor.file.ReadAt(buffer[offset:offset+length], *position)
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(filesystem, resolvePath(path))
}
