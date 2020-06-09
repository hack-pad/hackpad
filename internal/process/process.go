package process

import (
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
)

const userID = 0
const groupID = 0
const currentPID = 1
const currentParentPID = 0

var currentUMask = 0755
var currentWorkingDirectory = os.Getenv("PWD")

func Init() {
	process := js.Global().Get("process")
	interop.SetFunc(process, "getuid", geteuid)
	interop.SetFunc(process, "geteuid", geteuid)
	interop.SetFunc(process, "getgid", getegid)
	interop.SetFunc(process, "getegid", getegid)
	interop.SetFunc(process, "getgroups", getgroups)
	interop.SetFunc(process, "pid", pid)
	interop.SetFunc(process, "ppid", ppid)
	interop.SetFunc(process, "umask", umask)
	interop.SetFunc(process, "cwd", cwd)
	interop.SetFunc(process, "chdir", chdir)
	interop.SetInitialized("process")
}

func geteuid(args []js.Value) (interface{}, error) {
	return userID, nil
}

func getegid(args []js.Value) (interface{}, error) {
	return groupID, nil
}

func getgroups(args []js.Value) (interface{}, error) {
	return groupID, nil
}

func pid(args []js.Value) (interface{}, error) {
	return currentPID, nil
}

func ppid(args []js.Value) (interface{}, error) {
	return currentParentPID, nil
}

func umask(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return currentUMask, nil
	}
	oldUMask := currentUMask
	currentUMask = args[0].Int()
	return oldUMask, nil
}

func cwd(args []js.Value) (interface{}, error) {
	return currentWorkingDirectory, nil
}

func chdir(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("a new directory argument is required")
	}
	newCWD := args[0].String()
	info, err := fs.Stat(newCWD)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.Errorf("%s is not a directory", info.Name())
	}
	currentWorkingDirectory = args[0].String()
	return nil, nil
}
