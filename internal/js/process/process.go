package process

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/process"
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
	jsProcess.Set("pid", currentProcess.PID())
	jsProcess.Set("ppid", currentProcess.ParentPID())
	interop.SetFunc(jsProcess, "umask", umask)
	interop.SetFunc(jsProcess, "cwd", cwd)
	interop.SetFunc(jsProcess, "chdir", chdir)

	globals.Set("child_process", map[string]interface{}{})
	childProcess := globals.Get("child_process")
	interop.SetFunc(childProcess, "spawn", spawn)
	//interop.SetFunc(childProcess, "spawnSync", spawnSync) // TODO is there any way to run spawnSync so we don't hit deadlock?
	interop.SetFunc(childProcess, "wait", wait)
	interop.SetFunc(childProcess, "waitSync", waitSync)
}

func switchedContext(pid, ppid process.PID) {
	jsProcess.Set("pid", pid)
	jsProcess.Set("ppid", ppid)
}

/*
func environ() map[string]interface{} {
	env := make(map[string]interface{})
	for _, pair := range os.Environ() {
		equalsIndex := strings.IndexRune(pair, '=')
		if equalsIndex == -1 {
			env[pair] = ""
		} else {
			key, val := pair[:equalsIndex], pair[equalsIndex+1:]
			env[key] = val
		}
	}
	return env
}
*/

func Dump() interface{} {
	return process.Dump()
}
