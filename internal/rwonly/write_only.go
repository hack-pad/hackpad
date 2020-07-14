package rwonly

import (
	"os"
	"syscall"

	"github.com/spf13/afero"
)

type writeOnlyFile struct {
	afero.File
}

// WriteOnly returns a write-only file handle.
// NOTE: Behavior is undefined if a directory is passed. A write-only directory should error out early on.
func WriteOnly(file afero.File) afero.File {
	return &writeOnlyFile{file}
}

func (w *writeOnlyFile) writeErr(op string) error {
	return &os.PathError{Op: op, Path: w.Name(), Err: syscall.EBADF}
}

func (w *writeOnlyFile) Read(p []byte) (n int, err error) {
	return 0, w.writeErr("read")
}

func (w *writeOnlyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, w.writeErr("readat")
}

func (w *writeOnlyFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, &os.PathError{Op: "readdir", Path: w.Name(), Err: syscall.ENOTDIR}
}

func (w *writeOnlyFile) Readdirnames(n int) ([]string, error) {
	return nil, &os.PathError{Op: "readdirnames", Path: w.Name(), Err: syscall.ENOTDIR}
}
