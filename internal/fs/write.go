package fs

import (
	"io"

	"github.com/johnstarich/go-wasm/internal/interop"
)

func (f *FileDescriptors) Write(fd FID, buffer []byte, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.fidMap[fd]
	if fileDescriptor == nil {
		return 0, interop.BadFileNumber(fd)
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	if position != nil {
		_, err := fileDescriptor.file.Seek(*position, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	n, err = fileDescriptor.file.Write(buffer[offset : offset+length])
	if err == io.EOF {
		err = nil
	}
	return
}
