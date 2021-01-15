package fs

import (
	"io"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/interop"
)

func (f *FileDescriptors) Read(fd FID, buffer blob.Blob, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return 0, interop.BadFileNumber(fd)
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	var readBuf blob.Blob
	if position == nil {
		readBuf, n, err = blob.Read(fileDescriptor.file, length)
	} else {
		readBuf, n, err = blob.ReadAt(fileDescriptor.file, length, *position)
	}
	if err == io.EOF {
		err = nil
	}
	_, setErr := buffer.Set(readBuf, int64(offset))
	if err == nil && setErr != nil {
		err = setErr
	}
	return
}
