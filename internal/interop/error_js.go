// +build js

package interop

import (
	"fmt"
	"syscall/js"

	"github.com/pkg/errors"
)

func WrapAsJSError(err error, message string) error {
	return wrapAsJSError(err, message)
}

func wrapAsJSError(err error, message string, args ...js.Value) error {
	if err == nil {
		return nil
	}

	errMessage := errors.Wrap(err, message).Error()
	for _, arg := range args {
		errMessage += fmt.Sprintf("\n%v", arg)
	}

	val := js.ValueOf(map[string]interface{}{
		"message": js.ValueOf(errMessage),
		"code":    js.ValueOf(mapToErrNo(err, errMessage)),
	})
	return js.Error{Value: val}
}
