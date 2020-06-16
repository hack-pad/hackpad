package fs

import (
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

const minFD = 3

type FileDescriptors struct {
	lastFileDescriptorID *atomic.Uint64
	fileDescriptorNames  map[string]*fileDescriptor
	fileDescriptorIDs    map[uint64]*fileDescriptor
	fileDescriptorMu     sync.Mutex
}

type fileDescriptor struct {
	id        uint64
	file      afero.File
	openCount *atomic.Uint64
}

func NewFileDescriptors() *FileDescriptors {
	return &FileDescriptors{
		lastFileDescriptorID: atomic.NewUint64(minFD),
		fileDescriptorNames:  make(map[string]*fileDescriptor),
		fileDescriptorIDs:    make(map[uint64]*fileDescriptor),
	}
}

func (f *FileDescriptors) Open(path string, flags int, mode os.FileMode) (fd uint64, err error) {
	path = resolvePath(path)
	file, err := getFile(path, flags, mode)
	if err != nil {
		return 0, err
	}
	return f.openWithFile(file)
}

func (f *FileDescriptors) openWithFile(file afero.File) (uint64, error) {
	if f.fileDescriptorNames[file.Name()] == nil {
		f.fileDescriptorMu.Lock()
		if f.fileDescriptorNames[file.Name()] == nil {
			fd := &fileDescriptor{
				id:        f.lastFileDescriptorID.Inc() - 1,
				file:      file,
				openCount: atomic.NewUint64(0),
			}
			f.fileDescriptorNames[file.Name()] = fd
			f.fileDescriptorIDs[fd.id] = fd
		}
		f.fileDescriptorMu.Unlock()
	}
	descriptor := f.fileDescriptorNames[file.Name()]
	descriptor.openCount.Inc()
	return descriptor.id, nil
}

func getFile(absPath string, flags int, mode os.FileMode) (afero.File, error) {
	if absPath == "/dev/null" {
		return newNullFile(), nil
	}
	return filesystem.OpenFile(absPath, flags, mode)
}
func (f *FileDescriptors) Close(fd uint64) error {
	fileDescriptor := f.fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return errors.Errorf("unknown fd: %d", fd)
	}
	if fileDescriptor.openCount.Dec() == 0 {
		f.fileDescriptorMu.Lock()
		if fileDescriptor.openCount.Load() == 0 {
			delete(f.fileDescriptorIDs, fd)
			delete(f.fileDescriptorNames, fileDescriptor.file.Name())
		}
		f.fileDescriptorMu.Unlock()
		return fileDescriptor.file.Close()
	}
	return nil
}

func (f *FileDescriptors) Fstat(fd uint64) (os.FileInfo, error) {
	fileDescriptor := f.fileDescriptorIDs[fd]
	if fileDescriptor == nil {
		return nil, errors.Errorf("unknown fd: %d", fd)
	}
	return fileDescriptor.file.Stat()
}
