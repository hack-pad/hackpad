package interop

import (
	"fmt"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/jserror"
)

var (
	ErrNotImplemented = jserror.New("operation not supported", "ENOSYS")
)

func BadFileNumber(fd common.FID) error {
	return jserror.New(fmt.Sprintf("Bad file number %d", fd), "EBADF")
}

func BadFileErr(identifier string) error {
	return jserror.New(fmt.Sprintf("Bad file %q", identifier), "EBADF")
}
