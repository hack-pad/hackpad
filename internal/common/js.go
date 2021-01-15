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
	r := recover()
	if r == nil {
		return
	}
	switch val := r.(type) {
	case error:
		*err = val
	case js.Value:
		*err = js.Error{Value: val}
	default:
		*err = errors.Errorf("%+v", val)
	}
}
