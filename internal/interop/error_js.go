// +build js

package interop

import (
	"syscall/js"

	"github.com/pkg/errors"
)

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
