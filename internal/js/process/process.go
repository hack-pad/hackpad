// +build js

package process

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/process"
)

var jsProcess = js.Global().Get("process")

type processShim struct {
	process *process.Process
}

func Init(process *process.Process) {
	shim := processShim{process}

	err := process.Files().MkdirAll(process.WorkingDirectory(), 0750) // TODO move to parent initialization
	if err != nil {
		panic(err)
	}
	globals := js.Global()

	interop.SetFunc(jsProcess, "getuid", shim.geteuid)
	interop.SetFunc(jsProcess, "geteuid", shim.geteuid)
	interop.SetFunc(jsProcess, "getgid", shim.getegid)
	interop.SetFunc(jsProcess, "getegid", shim.getegid)
	interop.SetFunc(jsProcess, "getgroups", shim.getgroups)
	jsProcess.Set("pid", process.PID())
	jsProcess.Set("ppid", process.ParentPID())
	interop.SetFunc(jsProcess, "umask", shim.umask)
	interop.SetFunc(jsProcess, "cwd", shim.cwd)
	interop.SetFunc(jsProcess, "chdir", shim.chdir)

	globals.Set("child_process", map[string]interface{}{})
	childProcess := globals.Get("child_process")
	interop.SetFunc(childProcess, "spawn", shim.spawn)
	//interop.SetFunc(childProcess, "spawnSync", shim.spawnSync) // TODO is there any way to run spawnSync so we don't hit deadlock?
	interop.SetFunc(childProcess, "wait", shim.wait)
	interop.SetFunc(childProcess, "waitSync", shim.waitSync)
}

func (s processShim) Dump() interface{} {
	return process.Dump()
}
