package common

import (
	"fmt"
	"syscall/js"
)

type PID uint64

func (p PID) JSValue() js.Value {
	return js.ValueOf(uint64(p))
}

func (p PID) String() string {
	return fmt.Sprintf("%d", p)
}
