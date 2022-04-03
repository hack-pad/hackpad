// +build js

package process

import (
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func (s processShim) wait(args []js.Value) ([]interface{}, error) {
	ret, err := s.waitSync(args)
	return []interface{}{ret}, err
}

func (s processShim) waitSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	pid := process.PID(args[0].Int())
	waitStatus := new(syscall.WaitStatus)
	wpid, err := s.Wait(pid, waitStatus, 0, nil)
	return map[string]interface{}{
		"pid":      wpid,
		"exitCode": waitStatus.ExitStatus(),
	}, err
}

func (s processShim) Wait(pid process.PID, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid process.PID, err error) {
	// TODO support options and rusage
	exitCode, err := s.waiter.Wait(pid)
	if wstatus != nil {
		const (
			// defined in syscall.WaitStatus
			exitCodeShift = 8
			exitedMask    = 0x7F
		)
		status := 0
		status |= exitCode << exitCodeShift // exit code
		status |= exitedMask                // exited
		*wstatus = syscall.WaitStatus(status)
	}
	return pid, err
}
