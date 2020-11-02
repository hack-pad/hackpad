package common

import (
	"fmt"
)

type PID uint64

func (p PID) String() string {
	return fmt.Sprintf("%d", p)
}
