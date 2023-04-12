//go:build js
// +build js

package common

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func (f FID) JSValue() js.Value {
	return js.ValueOf(uint64(f))
}

func (p PID) JSValue() js.Value {
	return js.ValueOf(uint64(p))
}

func CatchException(err *error) {
	recoverErr := handleRecovery(recover())
	if recoverErr != nil {
		*err = recoverErr
	}
}

func CatchExceptionHandler(fn func(err error)) {
	err := handleRecovery(recover())
	if err != nil {
		fn(err)
	}
}

func handleRecovery(r interface{}) error {
	if r == nil {
		return nil
	}
	switch val := r.(type) {
	case error:
		return val
	case js.Value:
		return js.Error{Value: val}
	default:
		return errors.Errorf("%+v", val)
	}
}
