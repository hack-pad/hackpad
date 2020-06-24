// +build js,wasm

package syscall

import (
	"strings"
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

func StartProcess(name string, argv []string, attr *ProcAttr) (pid int, handle uintptr, err error) {
	if len(argv) == 0 {
		// ensure always at least 1 arg
		argv = []string{name}
	}
	jsArgs := make([]interface{}, 0, len(argv)-1) // JS args don't include the command name
	for _, arg := range argv[1:] {
		jsArgs = append(jsArgs, arg)
	}

	cwd := attr.Dir
	if cwd == "" {
		cwd, err = Getwd()
		if err != nil {
			return 0, 0, err
		}
	}
	var env map[string]interface{}
	if attr.Env != nil {
		env = splitEnvPairs(attr.Env)
	} else {
		env = splitEnvPairs(Environ())
	}

	var fds []interface{}
	for _, f := range attr.Files {
		fds = append(fds, f)
	}

	ret := jsChildProcess.Call("spawn", name, jsArgs, map[string]interface{}{
		"argv0": argv[0],
		"cwd":   attr.Dir,
		"env":   env,
		"stdio": fds,
	})
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

func splitEnvPairs(pairs []string) map[string]interface{} {
	env := make(map[string]interface{})
	for _, pair := range pairs {
		equalIndex := strings.IndexRune(pair, '=')
		if equalIndex == -1 {
			env[pair] = ""
		} else {
			key, value := pair[:equalIndex], pair[equalIndex+1:]
			env[key] = value
		}
	}
	return env
}
