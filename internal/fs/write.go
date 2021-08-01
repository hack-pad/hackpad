package fs

import (
	"io"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

func (f *FileDescriptors) Write(fd FID, buffer blob.Blob, offset, length int, position *int64) (n int, err error) {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return 0, interop.BadFileNumber(fd)
	}
	file, ok := fileDescriptor.file.(io.Writer)
	if !ok {
		return 0, hackpadfs.ErrNotImplemented
	}
	// 'offset' in Node.js's read is the offset in the buffer to start writing at,
	// and 'position' is where to begin reading from in the file.
	if position != nil {
		_, err := hackpadfs.SeekFile(fileDescriptor.file, *position, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}
	dataToCopy, err := blob.View(buffer, int64(offset), int64(offset+length))
	if err != nil {
		return 0, &hackpadfs.PathError{Op: "write", Path: fileDescriptor.openedName, Err: err}
	}
	n, err = blob.Write(file, dataToCopy)
	if err == io.EOF {
		err = nil
	}
	return
}
