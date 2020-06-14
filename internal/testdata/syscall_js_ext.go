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
	if jsFS.Get("flock").IsUndefined() {
		// fs.flock is unavailable on Node.js and JS by default
		// typically it's included via node-fs-ext
		return ENOSYS
	}

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
