//go:build js
// +build js

package process

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/process"
)

var jsProcess = js.Global().Get("process")

func Init() {
	process.Init(switchedContext)

	currentProcess := process.Current()
	err := currentProcess.Files().MkdirAll(currentProcess.WorkingDirectory(), 0750)
	if err != nil {
		panic(err)
	}
	globals := js.Global()

	interop.SetFunc(jsProcess, "getuid", geteuid)
	interop.SetFunc(jsProcess, "geteuid", geteuid)
	interop.SetFunc(jsProcess, "getgid", getegid)
	interop.SetFunc(jsProcess, "getegid", getegid)
	interop.SetFunc(jsProcess, "getgroups", getgroups)
	jsProcess.Set("pid", currentProcess.PID().JSValue())
	jsProcess.Set("ppid", currentProcess.ParentPID().JSValue())
	interop.SetFunc(jsProcess, "umask", umask)
	interop.SetFunc(jsProcess, "cwd", cwd)
	interop.SetFunc(jsProcess, "chdir", chdir)

	globals.Set("child_process", map[string]interface{}{})
	childProcess := globals.Get("child_process")
	interop.SetFunc(childProcess, "spawn", spawn)
	// interop.SetFunc(childProcess, "spawnSync", spawnSync) // TODO is there any way to run spawnSync so we don't hit deadlock?
	interop.SetFunc(childProcess, "wait", wait)
	interop.SetFunc(childProcess, "waitSync", waitSync)
}

func switchedContext(pid, ppid process.PID) {
	jsProcess.Set("pid", pid.JSValue())
	jsProcess.Set("ppid", ppid.JSValue())
}

func Dump() interface{} {
	return process.Dump()
}
