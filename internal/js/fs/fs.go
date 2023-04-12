//go:build js
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
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/hack-pad/hackpad/internal/promise"
)

/*
fchown(fd, uid, gid, callback) { callback(enosys()); },
lchown(path, uid, gid, callback) { callback(enosys()); },
link(path, link, callback) { callback(enosys()); },
readlink(path, callback) { callback(enosys()); },
symlink(path, link, callback) { callback(enosys()); },
truncate(path, length, callback) { callback(enosys()); },
*/

func Init() {
	fs := js.Global().Get("fs")
	constants := fs.Get("constants")
	constants.Set("O_RDONLY", syscall.O_RDONLY)
	constants.Set("O_WRONLY", syscall.O_WRONLY)
	constants.Set("O_RDWR", syscall.O_RDWR)
	constants.Set("O_CREAT", syscall.O_CREATE)
	constants.Set("O_TRUNC", syscall.O_TRUNC)
	constants.Set("O_APPEND", syscall.O_APPEND)
	constants.Set("O_EXCL", syscall.O_EXCL)
	interop.SetFunc(fs, "chmod", chmod)
	interop.SetFunc(fs, "chmodSync", chmodSync)
	interop.SetFunc(fs, "chown", chown)
	interop.SetFunc(fs, "chownSync", chownSync)
	interop.SetFunc(fs, "close", closeFn)
	interop.SetFunc(fs, "closeSync", closeSync)
	interop.SetFunc(fs, "fchmod", fchmod)
	interop.SetFunc(fs, "fchmodSync", fchmodSync)
	interop.SetFunc(fs, "flock", flock)
	interop.SetFunc(fs, "flockSync", flockSync)
	interop.SetFunc(fs, "fstat", fstat)
	interop.SetFunc(fs, "fstatSync", fstatSync)
	interop.SetFunc(fs, "fsync", fsync)
	interop.SetFunc(fs, "fsyncSync", fsyncSync)
	interop.SetFunc(fs, "ftruncate", ftruncate)
	interop.SetFunc(fs, "ftruncateSync", ftruncateSync)
	interop.SetFunc(fs, "lstat", lstat)
	interop.SetFunc(fs, "lstatSync", lstatSync)
	interop.SetFunc(fs, "mkdir", mkdir)
	interop.SetFunc(fs, "mkdirSync", mkdirSync)
	interop.SetFunc(fs, "open", open)
	interop.SetFunc(fs, "openSync", openSync)
	interop.SetFunc(fs, "pipe", pipe)
	interop.SetFunc(fs, "pipeSync", pipeSync)
	interop.SetFunc(fs, "read", read)
	interop.SetFunc(fs, "readSync", readSync)
	interop.SetFunc(fs, "readdir", readdir)
	interop.SetFunc(fs, "readdirSync", readdirSync)
	interop.SetFunc(fs, "rename", rename)
	interop.SetFunc(fs, "renameSync", renameSync)
	interop.SetFunc(fs, "rmdir", rmdir)
	interop.SetFunc(fs, "rmdirSync", rmdirSync)
	interop.SetFunc(fs, "stat", stat)
	interop.SetFunc(fs, "statSync", statSync)
	interop.SetFunc(fs, "unlink", unlink)
	interop.SetFunc(fs, "unlinkSync", unlinkSync)
	interop.SetFunc(fs, "utimes", utimes)
	interop.SetFunc(fs, "utimesSync", utimesSync)
	interop.SetFunc(fs, "write", write)
	interop.SetFunc(fs, "writeSync", writeSync)

	global.Set("getMounts", js.FuncOf(getMounts))
	global.Set("destroyMount", js.FuncOf(destroyMount))
	global.Set("overlayTarGzip", js.FuncOf(overlayTarGzip))
	global.Set("overlayIndexedDB", js.FuncOf(overlayIndexedDB))
	global.Set("dumpZip", js.FuncOf(dumpZip))

	// Set up system directories
	files := process.Current().Files()
	if err := files.MkdirAll(os.TempDir(), 0777); err != nil {
		panic(err)
	}
}

func Dump(basePath string) interface{} {
	basePath = common.ResolvePath(process.Current().WorkingDirectory(), basePath)
	return fs.Dump(basePath)
}

func dumpZip(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return interop.WrapAsJSError(errors.New("dumpZip: file path is required"), "EINVAL")
	}
	path := args[0].String()
	path = common.ResolvePath(process.Current().WorkingDirectory(), path)
	return interop.WrapAsJSError(fs.DumpZip(path), "dumpZip")
}

func getMounts(this js.Value, args []js.Value) interface{} {
	var mounts []string
	for _, p := range fs.Mounts() {
		mounts = append(mounts, p.Path)
	}
	return interop.SliceFromStrings(mounts)
}

func destroyMount(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return interop.WrapAsJSError(errors.New("destroyMount: mount path is required"), "EINVAL")
	}
	resolve, reject, prom := promise.New()
	mountPath := args[0].String()
	go func() {
		jsErr := interop.WrapAsJSError(fs.DestroyMount(mountPath), "destroyMount")
		if jsErr.Type() != js.TypeNull {
			reject(jsErr)
		} else {
			resolve(nil)
		}
	}()
	return prom
}
