package fs

import (
	"os"
	"sync"
	goAtomic "sync/atomic"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

const minFD = 3

type FileDescriptors struct {
	previousFID FID
	nameMap     map[string]*fileDescriptor
	fidMap      map[FID]*fileDescriptor
	mu          sync.Mutex
}

type fileDescriptor struct {
	id        FID
	file      afero.File
	openCount *atomic.Uint64
}

func NewFileDescriptors() *FileDescriptors {
	return &FileDescriptors{
		previousFID: minFD,
		nameMap:     make(map[string]*fileDescriptor),
		fidMap:      make(map[FID]*fileDescriptor),
	}
}

func (f *FileDescriptors) Open(path string, flags int, mode os.FileMode) (fd FID, err error) {
	path = resolvePath(path)
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
