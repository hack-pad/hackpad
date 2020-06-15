// +build js,wasm

package syscall

import (
	"syscall/js"
)

const (
	LOCK_SH = 0x1
	LOCK_EX = 0x2
	LOCK_UN = 0x8
)

var jsChildProcess = js.Global().Get("child_process")

func Flock(fd, how int) error {
	_, err := fsCall("flock", fd, how)
	return err
}

func StartProcess(argv0 string, argv []string, attr *ProcAttr) (pid int, handle uintptr, err error) {
	jsArgv := make([]interface{}, 0, len(argv))
	for _, arg := range argv {
		jsArgv = append(jsArgv, arg)
	}
	ret := jsChildProcess.Call("spawn", argv0, jsArgv)
	pid = ret.Get("pid").Int()
	return pid, 0, nil
}

func Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error) {
	if pid <= 0 {
		// waiting on any child process is not currently supported
		return -1, ENOSYS
	}
	// TODO support wstatus, options, and rusage
	ret, err := childProcessCall("wait", pid)
	return ret.Int(), err
}

func childProcessCall(name string, args ...interface{}) (js.Value, error) {
	type callResult struct {
		val js.Value
		err error
	}

	c := make(chan callResult, 1)
	jsChildProcess.Call(name, append(args, js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		var res callResult

		if jsErr := args[0]; !jsErr.IsNull() {
			res.err = mapJSError(jsErr)
		}

		res.val = js.Undefined()
		if len(args) >= 2 {
			res.val = args[1]
		}

		c <- res
		return nil
	}))...)
	res := <-c
	return res.val, res.err
}
