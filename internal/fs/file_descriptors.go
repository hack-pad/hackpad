package fs

import (
	"os"
	"path/filepath"
	"sync"
	goAtomic "sync/atomic"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

const minFD = 3

var (
	ErrNotDir = interop.NewError("not a directory", "ENOTDIR")

	filesystem = afero.NewMemMapFs()
)

type FileDescriptors struct {
	previousFID      FID
	nameMap          map[string]*fileDescriptor
	fidMap           map[FID]*fileDescriptor
	mu               sync.Mutex
	workingDirectory string
}

type fileDescriptor struct {
	id        FID
	file      afero.File
	openCount *atomic.Uint64
}

func NewFileDescriptors(workingDirectory string) (*FileDescriptors, func(wd string) error) {
	f := &FileDescriptors{
		previousFID:      minFD,
		nameMap:          make(map[string]*fileDescriptor),
		fidMap:           make(map[FID]*fileDescriptor),
		workingDirectory: workingDirectory,
	}
	return f, f.setWorkingDirectory
}

func (f *FileDescriptors) setWorkingDirectory(wd string) error {
	info, err := f.Stat(wd)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}
	f.mu.Lock()
	f.workingDirectory = wd
	f.mu.Unlock()
	return nil
}

func (f *FileDescriptors) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(f.workingDirectory, path)
}

func (f *FileDescriptors) Open(path string, flags int, mode os.FileMode) (fd FID, err error) {
	path = f.resolvePath(path)
	file, err := getFile(path, flags, mode)
	if err != nil {
		return 0, err
	}
	return f.openWithFile(file)
}

func (f *FileDescriptors) openWithFile(file afero.File) (FID, error) {
	if f.nameMap[file.Name()] == nil {
		f.mu.Lock()
		if f.nameMap[file.Name()] == nil {
			nextFID := goAtomic.AddUint64((*uint64)(&f.previousFID), 1)
			fd := &fileDescriptor{
				id:        FID(nextFID - 1),
				file:      file,
				openCount: atomic.NewUint64(0),
			}
			f.nameMap[file.Name()] = fd
			f.fidMap[fd.id] = fd
		}
		f.mu.Unlock()
	}
	descriptor := f.nameMap[file.Name()]
	descriptor.openCount.Inc()
	return descriptor.id, nil
}

func getFile(absPath string, flags int, mode os.FileMode) (afero.File, error) {
	if absPath == "/dev/null" {
		return newNullFile(), nil
	}
	return filesystem.OpenFile(absPath, flags, mode)
}

func (f *FileDescriptors) Close(fd FID) error {
	fileDescriptor := f.fidMap[fd]
	if fileDescriptor == nil {
		return errors.Errorf("unknown fd: %d", fd)
	}
	if fileDescriptor.openCount.Dec() == 0 {
		f.mu.Lock()
		if fileDescriptor.openCount.Load() == 0 {
			delete(f.fidMap, fd)
			delete(f.nameMap, fileDescriptor.file.Name())
		}
		f.mu.Unlock()
		return fileDescriptor.file.Close()
	}
	return nil
}

func (f *FileDescriptors) Fstat(fd FID) (os.FileInfo, error) {
	fileDescriptor := f.fidMap[fd]
	if fileDescriptor == nil {
		return nil, errors.Errorf("unknown fd: %d", fd)
	}
	return fileDescriptor.file.Stat()
}

func (f *FileDescriptors) ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(filesystem, f.resolvePath(path))
}

func (f *FileDescriptors) ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(filesystem, f.resolvePath(path))
}

func (f *FileDescriptors) RemoveDir(path string) error {
	path = f.resolvePath(path)
	info, err := f.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}
	return filesystem.Remove(path)
}

func (f *FileDescriptors) Chmod(path string, mode os.FileMode) error {
	return filesystem.Chmod(f.resolvePath(path), mode)
}

func (f *FileDescriptors) Stat(path string) (os.FileInfo, error) {
	return filesystem.Stat(f.resolvePath(path))
}

func (f *FileDescriptors) Lstat(path string) (os.FileInfo, error) {
	// TODO add proper symlink support
	return filesystem.Stat(f.resolvePath(path))
}

func (f *FileDescriptors) Mkdir(path string, mode os.FileMode) error {
	return filesystem.Mkdir(f.resolvePath(path), mode)
}

func (f *FileDescriptors) MkdirAll(path string, mode os.FileMode) error {
	return filesystem.MkdirAll(f.resolvePath(path), mode)
}

func (f *FileDescriptors) Unlink(path string) error {
	path = f.resolvePath(path)
	info, err := f.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.ErrPermission
	}
	return filesystem.Remove(path)
}

func (f *FileDescriptors) Utimes(path string, atime, mtime time.Time) error {
	return filesystem.Chtimes(f.resolvePath(path), atime, mtime)
}
