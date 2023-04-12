//go:build js
// +build js

package process

import (
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

func wait(args []js.Value) ([]interface{}, error) {
	ret, err := waitSync(args)
	return []interface{}{ret}, err
}

func waitSync(args []js.Value) (interface{}, error) {
	if len(args) != 1 {
		return nil, errors.Errorf("Invalid number of args, expected 1: %v", args)
	}
	pid := process.PID(args[0].Int())
	waitStatus := new(syscall.WaitStatus)
	wpid, err := Wait(pid, waitStatus, 0, nil)
	return js.ValueOf(map[string]interface{}{
		"pid":      wpid.JSValue(),
		"exitCode": waitStatus.ExitStatus(),
	}), err
}

func Wait(pid process.PID, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid process.PID, err error) {
	// TODO support options and rusage
	p, ok := process.Get(pid)
	if !ok {
		return 0, errors.Errorf("Unknown child process: %d", pid)
	}

	exitCode, err := p.Wait()
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
