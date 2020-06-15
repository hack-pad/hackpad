package process

import (
	"syscall"
	"syscall/js"

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
	pid := PID(args[0].Int())
	return Wait(pid, nil, 0, nil)
}

func Wait(pid PID, wstatus *syscall.WaitStatus, options int, rusage *syscall.Rusage) (wpid PID, err error) {
	// TODO support wait status, options, and rusage
	if process := pids[pid]; process != nil {
		return pid, process.Wait()
	}
	return 0, errors.Errorf("Unknown child process: %d", pid)
}
