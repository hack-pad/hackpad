//go:build js
// +build js

package interop

import (
	"fmt"
	"syscall/js"

	"github.com/pkg/errors"
)

func WrapAsJSError(err error, message string) js.Value {
	return wrapAsJSError(err, message)
}

func wrapAsJSError(err error, message string, args ...js.Value) js.Value {
	if err == nil {
		return js.Null()
	}

	errMessage := errors.Wrap(err, message).Error()
	for _, arg := range args {
		errMessage += fmt.Sprintf("\n%v", arg)
	}

	return js.ValueOf(map[string]interface{}{
		"message": js.ValueOf(errMessage),
		"code":    js.ValueOf(mapToErrNo(err, errMessage)),
	})
}
