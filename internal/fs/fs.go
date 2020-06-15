package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

/*
fchmod(fd, mode, callback) { callback(enosys()); },
fchown(fd, uid, gid, callback) { callback(enosys()); },
fsync(fd, callback) { callback(null); },
ftruncate(fd, length, callback) { callback(enosys()); },
lchown(path, uid, gid, callback) { callback(enosys()); },
link(path, link, callback) { callback(enosys()); },
readlink(path, callback) { callback(enosys()); },
rename(from, to, callback) { callback(enosys()); },
symlink(path, link, callback) { callback(enosys()); },
truncate(path, length, callback) { callback(enosys()); },
*/

var filesystem = afero.NewMemMapFs()

const minFD = 3

var lastFileDescriptorID = atomic.NewUint64(minFD)
var fileDescriptorNames = make(map[string]*fileDescriptor)
var fileDescriptorIDs = make(map[uint64]*fileDescriptor)
var fileDescriptorMu sync.Mutex

type fileDescriptor struct {
	id        uint64
	file      afero.File
	openCount *atomic.Uint64
}

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
	interop.SetFunc(fs, "flock", flock)
	interop.SetFunc(fs, "flockSync", flockSync)
	interop.SetFunc(fs, "fstat", fstat)
	interop.SetFunc(fs, "fstatSync", fstatSync)
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
}

func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(interop.WorkingDirectory(), path)
}

func Dump() interface{} {
	var total int64
	err := afero.Walk(filesystem, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return err
	}
	return fmt.Sprintf("%d bytes", total)
}
