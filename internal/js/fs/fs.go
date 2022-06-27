// +build js

package fs

import (
	"errors"
	"os"
	"syscall"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jserror"
	"github.com/hack-pad/hackpad/internal/jsfunc"
	"github.com/hack-pad/hackpad/internal/process"
)

type fileShim struct {
	process *process.Process
}

func Init(process *process.Process) {
	shim := fileShim{process}

	fs := js.Global().Get("fs")
	constants := fs.Get("constants")
	constants.Set("O_RDONLY", syscall.O_RDONLY)
	constants.Set("O_WRONLY", syscall.O_WRONLY)
	constants.Set("O_RDWR", syscall.O_RDWR)
	constants.Set("O_CREAT", syscall.O_CREATE)
	constants.Set("O_TRUNC", syscall.O_TRUNC)
	constants.Set("O_APPEND", syscall.O_APPEND)
	constants.Set("O_EXCL", syscall.O_EXCL)
	interop.SetFunc(fs, "chmod", shim.chmod)
	interop.SetFunc(fs, "chmodSync", shim.chmodSync)
	interop.SetFunc(fs, "chown", shim.chown)
	interop.SetFunc(fs, "chownSync", shim.chownSync)
	interop.SetFunc(fs, "close", shim.closeFn)
	interop.SetFunc(fs, "closeSync", shim.closeSync)
	interop.SetFunc(fs, "fchmod", shim.fchmod)
	interop.SetFunc(fs, "fchmodSync", shim.fchmodSync)
	interop.SetFunc(fs, "flock", shim.flock)
	interop.SetFunc(fs, "flockSync", shim.flockSync)
	interop.SetFunc(fs, "fstat", shim.fstat)
	interop.SetFunc(fs, "fstatSync", shim.fstatSync)
	interop.SetFunc(fs, "fsync", shim.fsync)
	interop.SetFunc(fs, "fsyncSync", shim.fsyncSync)
	interop.SetFunc(fs, "ftruncate", shim.ftruncate)
	interop.SetFunc(fs, "ftruncateSync", shim.ftruncateSync)
	interop.SetFunc(fs, "lstat", shim.lstat)
	interop.SetFunc(fs, "lstatSync", shim.lstatSync)
	interop.SetFunc(fs, "mkdir", shim.mkdir)
	interop.SetFunc(fs, "mkdirSync", shim.mkdirSync)
	interop.SetFunc(fs, "open", shim.open)
	interop.SetFunc(fs, "openSync", shim.openSync)
	interop.SetFunc(fs, "pipe", shim.pipe)
	interop.SetFunc(fs, "pipeSync", shim.pipeSync)
	interop.SetFunc(fs, "read", shim.read)
	interop.SetFunc(fs, "readSync", shim.readSync)
	interop.SetFunc(fs, "readdir", shim.readdir)
	interop.SetFunc(fs, "readdirSync", shim.readdirSync)
	interop.SetFunc(fs, "rename", shim.rename)
	interop.SetFunc(fs, "renameSync", shim.renameSync)
	interop.SetFunc(fs, "rmdir", shim.rmdir)
	interop.SetFunc(fs, "rmdirSync", shim.rmdirSync)
	interop.SetFunc(fs, "stat", shim.stat)
	interop.SetFunc(fs, "statSync", shim.statSync)
	interop.SetFunc(fs, "unlink", shim.unlink)
	interop.SetFunc(fs, "unlinkSync", shim.unlinkSync)
	interop.SetFunc(fs, "utimes", shim.utimes)
	interop.SetFunc(fs, "utimesSync", shim.utimesSync)
	interop.SetFunc(fs, "write", shim.write)
	interop.SetFunc(fs, "writeSync", shim.writeSync)

	global.Set("getMounts", jsfunc.Promise(shim.getMounts))
	global.Set("destroyMount", jsfunc.Promise(shim.destroyMount))
	global.Set("overlayTarGzip", jsfunc.Promise(shim.overlayTarGzip))
	global.Set("overlayIndexedDB", jsfunc.Promise(shim.overlayIndexedDB))
	global.Set("dumpZip", jsfunc.Promise(shim.dumpZip))

	// Set up system directories
	files := process.Files()
	if err := files.MkdirAll(os.TempDir(), 0777); err != nil {
		panic(err)
	}
}

func (s fileShim) Dump(basePath string) interface{} {
	basePath = common.ResolvePath(s.process.WorkingDirectory(), basePath)
	return fs.Dump(basePath)
}

func (s fileShim) dumpZip(this js.Value, args []js.Value) (js.Wrapper, error) {
	if len(args) != 1 {
		return nil, jserror.Wrap(errors.New("dumpZip: file path is required"), "EINVAL")
	}
	path := args[0].String()
	path = common.ResolvePath(s.process.WorkingDirectory(), path)
	return nil, jserror.Wrap(fs.DumpZip(path), "dumpZip")
}

func (s fileShim) getMounts(this js.Value, args []js.Value) (js.Wrapper, error) {
	var mounts []string
	for _, p := range fs.Mounts() {
		mounts = append(mounts, p.Path)
	}
	return interop.SliceFromStrings(mounts), nil
}

func (s fileShim) destroyMount(this js.Value, args []js.Value) (js.Wrapper, error) {
	if len(args) < 1 {
		return nil, jserror.Wrap(errors.New("destroyMount: mount path is required"), "EINVAL")
	}
	mountPath := args[0].String()
	return nil, jserror.Wrap(fs.DestroyMount(mountPath), "destroyMount")
}
