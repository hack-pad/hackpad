package jsworker

import (
	"syscall/js"

	"github.com/pkg/errors"
)

func jsInt(v js.Value) int {
	if v.Type() != js.TypeNumber {
		return 0
	}
	return v.Int()
}

func jsString(v js.Value) string {
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}

func jsBool(v js.Value) bool {
	if v.Type() != js.TypeBoolean {
		return false
	}
	return v.Bool()
}

func jsError(v js.Value) error {
	if !v.Truthy() {
		return nil
	}
	return errors.Errorf("%v", v)
}
