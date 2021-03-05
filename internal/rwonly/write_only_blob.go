package rwonly

import (
	"os"
	"syscall"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/spf13/afero"
)

type blobFileWriter interface {
	afero.File
	blob.Writer
	blob.WriterAt
}

type blobWriteOnlyFile struct {
	blobFileWriter
}

func BlobWriteOnly(file afero.File) afero.File {
	if blobFile, ok := file.(blobFileWriter); ok {
		return &blobWriteOnlyFile{blobFile}
	}
	return WriteOnly(file)
}

func (w *blobWriteOnlyFile) readErr(op string) error {
	return &os.PathError{Op: op, Path: w.Name(), Err: syscall.EBADF}
}

func (w *blobWriteOnlyFile) Read(p []byte) (n int, err error) {
	return 0, w.readErr("read")
}

func (w *blobWriteOnlyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, w.readErr("readat")
}
