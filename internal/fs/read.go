package fs

import (
	"io"

	"github.com/pkg/errors"
)

func (f *FileDescriptors) Read(fd uint64, buffer []byte, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.fileDescriptorIDs[fd]
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
