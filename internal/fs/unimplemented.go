package fs

import (
	"os"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
)

var _ hackpadfs.File = &unimplementedFile{}

// unimplementedFile can be embedded in special files like /dev/null to provide a default unimplemented hackpadfs.File interface
type unimplementedFile struct{}

func (f unimplementedFile) Close() error                     { return interop.ErrNotImplemented }
func (f unimplementedFile) Read(p []byte) (n int, err error) { return 0, interop.ErrNotImplemented }
func (f unimplementedFile) Stat() (os.FileInfo, error)       { return nil, interop.ErrNotImplemented }
