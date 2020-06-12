package fs

import (
	"os"
	"sync"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

/*
chmod(path, mode, callback) { callback(enosys()); },
chown(path, uid, gid, callback) { callback(enosys()); },
fchmod(fd, mode, callback) { callback(enosys()); },
fchown(fd, uid, gid, callback) { callback(enosys()); },
fsync(fd, callback) { callback(null); },
ftruncate(fd, length, callback) { callback(enosys()); },
lchown(path, uid, gid, callback) { callback(enosys()); },
link(path, link, callback) { callback(enosys()); },
readlink(path, callback) { callback(enosys()); },
rename(from, to, callback) { callback(enosys()); },
rmdir(path, callback) { callback(enosys()); },
symlink(path, link, callback) { callback(enosys()); },
truncate(path, length, callback) { callback(enosys()); },
unlink(path, callback) { callback(enosys()); },
utimes(path, atime, mtime, callback) { callback(enosys()); },
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
	constants.Set("O_RDONLY", os.O_RDONLY)
	constants.Set("O_WRONLY", os.O_WRONLY)
	constants.Set("O_RDWR", os.O_RDWR)
	constants.Set("O_CREAT", os.O_CREATE)
	constants.Set("O_TRUNC", os.O_TRUNC)
	constants.Set("O_APPEND", os.O_APPEND)
	constants.Set("O_EXCL", os.O_EXCL)
	interop.SetFunc(fs, "close", closeFn)
	interop.SetFunc(fs, "closeSync", closeSync)
	interop.SetFunc(fs, "fstat", fstat)
	interop.SetFunc(fs, "fstatSync", fstatSync)
	interop.SetFunc(fs, "lstat", lstat)
	interop.SetFunc(fs, "lstatSync", lstatSync)
	interop.SetFunc(fs, "mkdir", mkdir)
	interop.SetFunc(fs, "mkdirSync", mkdirSync)
	interop.SetFunc(fs, "open", open)
	interop.SetFunc(fs, "openSync", openSync)
	interop.SetFunc(fs, "read", read)
	interop.SetFunc(fs, "readSync", readSync)
	interop.SetFunc(fs, "readdir", readdir)
	interop.SetFunc(fs, "readdirSync", readdirSync)
	interop.SetFunc(fs, "stat", stat)
	interop.SetFunc(fs, "statSync", statSync)
	interop.SetFunc(fs, "write", write)
	interop.SetFunc(fs, "writeSync", writeSync)
}
