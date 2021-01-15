package blob

import "io"

type Reader interface {
	ReadBlob(length int) (blob Blob, n int, err error)
}

type ReaderAt interface {
	ReadBlobAt(length int, off int64) (blob Blob, n int, err error)
}

type Writer interface {
	WriteBlob(p Blob) (n int, err error)
}

type WriterAt interface {
	WriteBlobAt(p Blob, off int64) (n int, err error)
}

func Read(r io.Reader, length int) (blob Blob, n int, err error) {
	if blobReader, ok := r.(Reader); ok {
		return blobReader.ReadBlob(length)
	}
	buf := make([]byte, length)
	n, err = r.Read(buf)
	return NewFromBytes(buf), n, err
}

func ReadAt(r io.ReaderAt, length int, off int64) (blob Blob, n int, err error) {
	if blobReader, ok := r.(ReaderAt); ok {
		return blobReader.ReadBlobAt(length, off)
	}
	buf := make([]byte, length)
	n, err = r.ReadAt(buf, off)
	return NewFromBytes(buf), n, err
}

func Write(w io.Writer, b Blob) (n int, err error) {
	if blobWriter, ok := w.(Writer); ok {
		return blobWriter.WriteBlob(b)
	}
	return w.Write(b.Bytes())
}

func WriteAt(w io.WriterAt, b Blob, off int64) (n int, err error) {
	if blobWriter, ok := w.(WriterAt); ok {
		return blobWriter.WriteBlobAt(b, off)
	}
	return w.WriteAt(b.Bytes(), off)
}
