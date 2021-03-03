// +build js

package rwonly

import (
	"os"
	"syscall"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/spf13/afero"
)

type blobFileReader interface {
	afero.File
	blob.Reader
	blob.ReaderAt
}

type blobReadOnlyFile struct {
	blobFileReader
}

func BlobReadOnly(file afero.File) afero.File {
	if blobFile, ok := file.(blobFileReader); ok {
		return &blobReadOnlyFile{blobFile}
	}
	return ReadOnly(file)
}

func (r *blobReadOnlyFile) writeErr(op string) error {
	return &os.PathError{Op: op, Path: r.Name(), Err: syscall.EBADF}
}

func (r *blobReadOnlyFile) Write(p []byte) (n int, err error) {
	return 0, r.writeErr("write")
}

func (r *blobReadOnlyFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, r.writeErr("writeat")
}

func (r *blobReadOnlyFile) Truncate(size int64) error {
	return r.writeErr("truncate")
}

func (r *blobReadOnlyFile) WriteString(s string) (ret int, err error) {
	return 0, r.writeErr("write")
}
