package interop

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

var (
	ErrNotImplemented = NewError("operation not supported", "ENOSYS")
)

type Error interface {
	error
	Message() string
	Code() string
}

type interopErr struct {
	error
	code string
}

func NewError(message, code string) Error {
	return WrapErr(errors.New(message), code)
}

func WrapErr(err error, code string) Error {
	return &interopErr{
		error: err,
		code:  code,
	}
}

func (e *interopErr) Message() string {
	return e.Error()
}

func (e *interopErr) Code() string {
	return e.code
}

func WrapAsJSError(err error, message string) error {
	if err == nil {
		return nil
	}

	val := js.ValueOf(map[string]interface{}{
		"message": js.ValueOf(errors.Wrap(err, message).Error()),
		"code":    js.ValueOf(mapToErrNo(err)),
	})
	return js.Error{Value: val}
}

// errno names pulled from syscall/tables_js.go
func mapToErrNo(err error) string {
	if err, ok := err.(Error); ok {
		return err.Code()
	}
	if err, ok := err.(interface{ Unwrap() error }); ok {
		return mapToErrNo(err.Unwrap())
	}
	switch err {
	case io.EOF, os.ErrNotExist, exec.ErrNotFound:
		return "ENOENT"
	case os.ErrExist:
		return "EEXIST"
	case os.ErrPermission:
		return "EPERM"
	case syscall.EISDIR:
		return "EISDIR"
	case syscall.ENOTDIR:
		return "ENOTDIR"
	}
	switch err.Error() {
	case os.ErrClosed.Error(), afero.ErrFileClosed.Error():
		return "EBADF" // if it was already closed, then the file descriptor was invalid
	}
	switch {
	case os.IsNotExist(err):
		return "ENOENT"
	case os.IsExist(err):
		return "EEXIST"
	default:
		log.Errorf("Unknown error type: (%T) %+v", err, err)
		return "EPERM"
	}
}

func BadFileNumber(fd common.FID) error {
	return NewError(fmt.Sprintf("Bad file number %d", fd), "EBADF")
}

func BadFileErr(identifier string) error {
	return NewError(fmt.Sprintf("Bad file %q", identifier), "EBADF")
}
