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
