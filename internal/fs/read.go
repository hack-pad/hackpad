package fs

import (
	"io"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
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
		readerAt, ok := fileDescriptor.file.(io.ReaderAt)
		if ok {
			readBuf, n, err = blob.ReadAt(readerAt, length, *position)
		} else {
			err = &hackpadfs.PathError{Op: "read", Path: fileDescriptor.openedName, Err: hackpadfs.ErrNotImplemented}
		}
	}
	if err == io.EOF {
		err = nil
	}
	if readBuf != nil {
		_, setErr := blob.Set(buffer, readBuf, int64(offset))
		if err == nil && setErr != nil {
			err = &hackpadfs.PathError{Op: "read", Path: fileDescriptor.openedName, Err: setErr}
		}
	}
	return
}

func (f *FileDescriptors) ReadFile(path string) (blob.Blob, error) {
	fd, err := f.Open(path, 0, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close(fd)

	info, err := f.Fstat(fd)
	if err != nil {
		return nil, err
	}

	buf, err := newBlobLength(int(info.Size()))
	if err != nil {
		return nil, err
	}
	_, err = f.Read(fd, buf, 0, buf.Len(), nil)
	return buf, err
}
