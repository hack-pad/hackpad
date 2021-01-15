package fs

import (
	"io"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/interop"
)

func (f *FileDescriptors) Write(fd FID, buffer blob.Blob, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.files[fd]
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
	dataToCopy, err := buffer.Slice(int64(offset), int64(offset+length))
	if err != nil {
		return 0, err
	}
	n, err = blob.Write(fileDescriptor.file, dataToCopy)
	if err == io.EOF {
		err = nil
	}
	return
}
