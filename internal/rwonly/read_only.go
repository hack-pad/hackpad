package rwonly

import (
	"os"
	"syscall"

	"github.com/spf13/afero"
)

type readOnlyFile struct {
	afero.File
}

func ReadOnly(file afero.File) afero.File {
	return &readOnlyFile{file}
}

func (r *readOnlyFile) writeErr(op string) error {
	return &os.PathError{Op: op, Path: r.Name(), Err: syscall.EBADF}
}

func (r *readOnlyFile) Write(p []byte) (n int, err error) {
	return 0, r.writeErr("write")
}

func (r *readOnlyFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, r.writeErr("writeat")
}

func (r *readOnlyFile) Truncate(size int64) error {
	return r.writeErr("truncate")
}

func (r *readOnlyFile) WriteString(s string) (ret int, err error) {
	return 0, r.writeErr("write")
}
