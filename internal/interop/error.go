package interop

import "errors"

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
