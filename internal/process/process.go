package process

import (
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

func Init() {
	err := fs.MkdirAll(interop.WorkingDirectory(), 0750)
	if err != nil {
		panic(err)
	}

	process := js.Global().Get("process")
	interop.SetFunc(process, "getuid", geteuid)
	interop.SetFunc(process, "geteuid", geteuid)
	interop.SetFunc(process, "getgid", getegid)
	interop.SetFunc(process, "getegid", getegid)
	interop.SetFunc(process, "getgroups", getgroups)
	process.Set("pid", currentPID)
	process.Set("ppid", currentParentPID)
	interop.SetFunc(process, "umask", umask)
	interop.SetFunc(process, "cwd", cwd)
	interop.SetFunc(process, "chdir", chdir)
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

func umask(args []js.Value) (interface{}, error) {
	if len(args) == 0 {
		return currentUMask, nil
	}
	oldUMask := currentUMask
	currentUMask = args[0].Int()
	return oldUMask, nil
}

func cwd(args []js.Value) (interface{}, error) {
	return interop.WorkingDirectory(), nil
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
	interop.SetWorkingDirectory(args[0].String())
	return nil, nil
}
