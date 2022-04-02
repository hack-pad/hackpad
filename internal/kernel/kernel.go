package kernel

import (
	"github.com/hack-pad/hackpad/internal/common"
	"go.uber.org/atomic"
)

const (
	minPID = 1
)

var (
	lastPID = atomic.NewUint64(minPID)
)

func ReservePID() common.PID {
	return common.PID(lastPID.Inc())
}
