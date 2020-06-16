package fs

import (
	"os"

	"github.com/johnstarich/go-wasm/internal/interop"
)

// unimplementedFile can be embedded in special files like /dev/null to provide a default unimplemented afero.File interface
type unimplementedFile struct{}

func (f unimplementedFile) Close() error                     { return interop.ErrNotImplemented }
func (f unimplementedFile) Read(p []byte) (n int, err error) { return 0, interop.ErrNotImplemented }
func (f unimplementedFile) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, interop.ErrNotImplemented
}
func (f unimplementedFile) Seek(offset int64, whence int) (int64, error) {
	return 0, interop.ErrNotImplemented
}
func (f unimplementedFile) Write(p []byte) (n int, err error) { return 0, interop.ErrNotImplemented }
func (f unimplementedFile) WriteAt(p []byte, off int64) (n int, err error) {
	return 0, interop.ErrNotImplemented
}

func (f unimplementedFile) Name() string { return "" }
func (f unimplementedFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, interop.ErrNotImplemented
}
func (f unimplementedFile) Readdirnames(n int) ([]string, error) {
	return nil, interop.ErrNotImplemented
}
func (f unimplementedFile) Stat() (os.FileInfo, error) { return nil, interop.ErrNotImplemented }
func (f unimplementedFile) Sync() error                { return interop.ErrNotImplemented }
func (f unimplementedFile) Truncate(size int64) error  { return interop.ErrNotImplemented }
func (f unimplementedFile) WriteString(s string) (ret int, err error) {
	return 0, interop.ErrNotImplemented
}
