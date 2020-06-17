package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	goAtomic "sync/atomic"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

var (
	ErrNotDir = interop.NewError("not a directory", "ENOTDIR")

	filesystem = afero.NewMemMapFs()
)

type FileDescriptors struct {
	parentPID        uint64
	previousFID      FID
	nameMap          map[string]*fileDescriptor
	fidMap           map[FID]*fileDescriptor
	mu               sync.Mutex
	workingDirectory *atomic.String
}

type fileDescriptor struct {
	id        FID
	file      afero.File
	openCount *atomic.Uint64
}

func (fd *fileDescriptor) String() string {
	switch file := fd.file.(type) {
	case *pipeWriteOnly:
		return fmt.Sprintf("%15s [%d] closed=%t, pids=%v, done=%v", fd.file.Name(), fd.id, fd.openCount.Load() == 0, pidBoolMapKeys(file.processOpened), pidBoolMapKeys(file.processClosed))
	case *pipeReadOnly:
		return fmt.Sprintf("%15s [%d] closed=%t, pids=%v, done=%v", fd.file.Name(), fd.id, fd.openCount.Load() == 0, pidBoolMapKeys(file.processOpened), pidBoolMapKeys(file.processClosed))
	default:
		return fmt.Sprintf("%15s [%d] closed=%t", fd.file.Name(), fd.id, fd.openCount.Load() == 0)
	}
}

func NewStdFileDescriptors(parentPID uint64, workingDirectory string) (*FileDescriptors, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		nameMap:          make(map[string]*fileDescriptor),
		fidMap:           make(map[FID]*fileDescriptor),
		workingDirectory: atomic.NewString(workingDirectory),
	}
	// order matters
	_, err := f.Open("/dev/stdin", syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	_, err = f.Open("/dev/stdout", syscall.O_WRONLY, 0)
	if err != nil {
		return nil, err
	}
	_, err = f.Open("/dev/stderr", syscall.O_WRONLY, 0)
	return f, err
}

func NewFileDescriptors(parentPID uint64, workingDirectory string, parentFiles *FileDescriptors, inheritFDs []*FID) (*FileDescriptors, func(wd string) error, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		nameMap:          make(map[string]*fileDescriptor),
		fidMap:           make(map[FID]*fileDescriptor),
		workingDirectory: atomic.NewString(workingDirectory),
	}
	if len(inheritFDs) == 0 {
		inheritFDs = []*FID{ptr(0), ptr(1), ptr(2)}
	}
	if len(inheritFDs) < 3 {
		return nil, nil, errors.Errorf("Invalid number of inherited file descriptors, must be 0 or at least 3: %#v", inheritFDs)
	}
	for _, fidPtr := range inheritFDs {
		if fidPtr == nil {
			return nil, nil, errors.New("Ignored file descriptors are unsupported") // TODO be sure to align FDs properly when skipping iterations
		}

		parentFD := parentFiles.fidMap[*fidPtr]
		if parentFD == nil {
			return nil, nil, errors.Errorf("Invalid parent FID %d", *fidPtr)
		}
		_, err := f.openWithFile(parentFD.file, true)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Failed to inherit file from parent process")
		}
	}
	return f, f.setWorkingDirectory, nil
}

func (f *FileDescriptors) setWorkingDirectory(path string) error {
	path = f.resolvePath(path)
	info, err := f.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}
	f.workingDirectory.Store(path)
	return nil
}

func (f *FileDescriptors) WorkingDirectory() string {
	return f.workingDirectory.Load()
}

func (f *FileDescriptors) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(f.workingDirectory.Load(), path)
}

func (f *FileDescriptors) Open(path string, flags int, mode os.FileMode) (fd FID, err error) {
	path = f.resolvePath(path)
	var file afero.File
	if fd, ok := f.nameMap[path]; ok {
		file = fd.file
	} else {
		file, err = getFile(path, flags, mode)
	}
	if err != nil {
		return 0, err
	}
	return f.openWithFile(file, false)
}

func (f *FileDescriptors) openWithFile(file afero.File, forceNewFID bool) (FID, error) {
	if f.nameMap[file.Name()] == nil || forceNewFID {
		f.mu.Lock()
		if f.nameMap[file.Name()] == nil || forceNewFID {
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
	switch pipe := file.(type) {
	case *pipeReadOnly:
		pipe.OpenPID(f.parentPID)
	case *pipeWriteOnly:
		pipe.OpenPID(f.parentPID)
	}
	return descriptor.id, nil
}

func getFile(absPath string, flags int, mode os.FileMode) (afero.File, error) {
	switch absPath {
	case "/dev/null":
		return newNullFile("/dev/null"), nil
	case "/dev/stdin":
		return newNullFile("/dev/stdin"), nil // TODO can this be mocked?
	case "/dev/stdout":
		return stdout, nil
	case "/dev/stderr":
		return stderr, nil
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
			//delete(f.fidMap, fd) // TODO is it safe to leave the old FD's hanging around? they're useful for debugging
			delete(f.nameMap, fileDescriptor.file.Name())
		}
		f.mu.Unlock()
		switch file := fileDescriptor.file.(type) {
		case *pipeReadOnly:
			return file.ClosePID(f.parentPID)
		case *pipeWriteOnly:
			return file.ClosePID(f.parentPID)
		default:
			return file.Close()
		}
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

func ptr(f FID) *FID {
	return &f
}

func pidBoolMapKeys(m map[uint64]bool) []int {
	var keys []int
	for pid := range m {
		keys = append(keys, int(pid))
	}
	sort.Ints(keys)
	return keys
}

func (f *FileDescriptors) String() string {
	var s strings.Builder
	var fids []FID
	for fid := range f.fidMap {
		fids = append(fids, fid)
	}
	sort.SliceStable(fids, func(a, b int) bool {
		return fids[a] < fids[b]
	})
	for _, fid := range fids {
		s.WriteString(f.fidMap[fid].String() + "\n")
	}
	return s.String()
}
