package fs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	goAtomic "sync/atomic"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/common"
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
	parentPID        common.PID
	previousFID      FID
	files            map[FID]*fileDescriptor
	mu               sync.Mutex
	workingDirectory *atomic.String
}

func NewStdFileDescriptors(parentPID common.PID, workingDirectory string) (*FileDescriptors, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		files:            make(map[FID]*fileDescriptor),
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

func NewFileDescriptors(parentPID common.PID, workingDirectory string, parentFiles *FileDescriptors, inheritFDs []*FID) (*FileDescriptors, func(wd string) error, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		files:            make(map[FID]*fileDescriptor),
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

		parentFD := parentFiles.files[*fidPtr]
		if parentFD == nil {
			return nil, nil, errors.Errorf("Invalid parent FID %d", *fidPtr)
		}
		fid := f.newFID()
		fd := parentFD.Dup(fid)
		f.addFileDescriptor(fd)
		fd.Open(parentPID)
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

func (f *FileDescriptors) newFID() FID {
	nextFID := goAtomic.AddUint64((*uint64)(&f.previousFID), 1)
	return FID(nextFID - 1)
}

func (f *FileDescriptors) Open(path string, flags int, mode os.FileMode) (fd FID, err error) {
	path = f.resolvePath(path)

	f.mu.Lock()
	defer f.mu.Unlock()
	descriptor, err := NewFileDescriptor(f.newFID(), path, flags, mode)
	if err != nil {
		return 0, err
	}
	f.addFileDescriptor(descriptor)
	descriptor.Open(f.parentPID)
	return descriptor.id, nil
}

func (f *FileDescriptors) addFileDescriptor(descriptor *fileDescriptor) {
	f.files[descriptor.id] = descriptor
}

func (f *FileDescriptors) removeFileDescriptor(descriptor *fileDescriptor) {
	delete(f.files, descriptor.id) // TODO is it safe to leave the old FD's hanging around? they're useful for debugging
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
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return interop.BadFileNumber(fd)
	}
	return fileDescriptor.Close(f.parentPID, &f.mu, func() {
		f.removeFileDescriptor(fileDescriptor)
	})
}

func (f *FileDescriptors) CloseAll() {
	f.mu.Lock()
	for _, fd := range f.files {
		_ = fd.closeAll(f.parentPID)
	}
	f.mu.Unlock()
}

func (f *FileDescriptors) Fstat(fd FID) (os.FileInfo, error) {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return nil, interop.BadFileNumber(fd)
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

func (f *FileDescriptors) String() string {
	var s strings.Builder
	var fids []FID
	for fid := range f.files {
		fids = append(fids, fid)
	}
	sort.SliceStable(fids, func(a, b int) bool {
		return fids[a] < fids[b]
	})
	for _, fid := range fids {
		s.WriteString(f.files[fid].String() + "\n")
	}
	return s.String()
}
