package jserror

import (
	"io"
	"os/exec"

	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

type Error interface {
	error
	Message() string
	Code() string
}

type jsErr struct {
	error
	code string
}

func New(message, code string) Error {
	return WrapErr(errors.New(message), code)
}

func WrapErr(err error, code string) Error {
	return &jsErr{
		error: err,
		code:  code,
	}
}

func (e *jsErr) Message() string {
	return e.Error()
}

func (e *jsErr) Code() string {
	return e.code
}

// errno names pulled from syscall/tables_js.go
func mapToErrNo(err error, debugMessage string) string {
	if err, ok := err.(Error); ok {
		return err.Code()
	}
	if err, ok := err.(interface{ Unwrap() error }); ok {
		return mapToErrNo(err.Unwrap(), debugMessage)
	}
	switch err {
	case io.EOF, exec.ErrNotFound:
		return "ENOENT"
	}
	switch {
	case errors.Is(err, hackpadfs.ErrClosed):
		return "EBADF" // if it was already closed, then the file descriptor was invalid
	case errors.Is(err, hackpadfs.ErrNotExist):
		return "ENOENT"
	case errors.Is(err, hackpadfs.ErrExist):
		return "EEXIST"
	case errors.Is(err, hackpadfs.ErrIsDir):
		return "EISDIR"
	case errors.Is(err, hackpadfs.ErrPermission):
		return "EPERM"
	default:
		log.Errorf("Unknown error type: (%T) %+v\n\n%s", err, err, debugMessage)
		return "EPERM"
	}
}
