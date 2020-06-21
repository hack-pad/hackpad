package fs

import (
	"io"

	"github.com/johnstarich/go-wasm/internal/interop"
)

func (f *FileDescriptors) Read(fd FID, buffer []byte, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.fidMap[fd]
	if fileDescriptor == nil {
		return 0, interop.BadFileNumber(fd)
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
