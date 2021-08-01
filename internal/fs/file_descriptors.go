package fs

import (
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	goAtomic "sync/atomic"
	"syscall"
	"time"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

var (
	ErrNotDir = interop.NewError("not a directory", "ENOTDIR")
)

type FileDescriptors struct {
	parentPID        common.PID
	previousFID      FID
	files            map[FID]*fileDescriptor
	mu               sync.Mutex
	workingDirectory *workingDirectory
}

func NewStdFileDescriptors(parentPID common.PID, workingDirectory string) (*FileDescriptors, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		files:            make(map[FID]*fileDescriptor),
		workingDirectory: newWorkingDirectory(workingDirectory),
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

func NewFileDescriptors(parentPID common.PID, workingDirectory string, parentFiles *FileDescriptors, inheritFDs []Attr) (*FileDescriptors, func(wd string) error, error) {
	f := &FileDescriptors{
		parentPID:        parentPID,
		previousFID:      0,
		files:            make(map[FID]*fileDescriptor),
		workingDirectory: newWorkingDirectory(workingDirectory),
	}
	if len(inheritFDs) == 0 {
		inheritFDs = []Attr{{FID: 0}, {FID: 1}, {FID: 2}}
	}
	if len(inheritFDs) < 3 {
		return nil, nil, errors.Errorf("Invalid number of inherited file descriptors, must be 0 or at least 3: %#v", inheritFDs)
	}
	for _, attr := range inheritFDs {
		var inheritFD FID
		switch {
		case attr.Ignore:
			return nil, nil, errors.New("Ignored file descriptors are unsupported") // TODO be sure to align FDs properly when skipping iterations
		case attr.Pipe:
			return nil, nil, errors.New("Pipe file descriptors are unsupported") // TODO align FDs like Ignore, but child FIDs on stdio property must be different than the real FIDs (see node docs)
		default:
			inheritFD = attr.FID
		}
		parentFD := parentFiles.files[inheritFD]
		if parentFD == nil {
			return nil, nil, errors.Errorf("Invalid parent FID %d", attr.FID)
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
	return f.workingDirectory.Set(path)
}

func (f *FileDescriptors) WorkingDirectory() string {
	wd, err := f.workingDirectory.Get()
	if err != nil {
		panic(err)
	}
	return path.Join("/", wd)
}

func (f *FileDescriptors) resolvePath(path string) string {
	return common.ResolvePath(f.WorkingDirectory(), path)
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

func getFile(absPath string, flags int, mode os.FileMode) (hackpadfs.File, error) {
	switch absPath {
	case "dev/null":
		return newNullFile("dev/null"), nil
	case "dev/stdin":
		return newNullFile("dev/stdin"), nil // TODO can this be mocked?
	case "dev/stdout":
		return stdout, nil
	case "dev/stderr":
		return stderr, nil
	}
	return hackpadfs.OpenFile(filesystem, absPath, flags, mode)
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

func (f *FileDescriptors) ReadDir(path string) ([]hackpadfs.DirEntry, error) {
	return hackpadfs.ReadDir(filesystem, f.resolvePath(path))
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
	return hackpadfs.Remove(filesystem, path)
}

func (f *FileDescriptors) Chmod(path string, mode os.FileMode) error {
	return hackpadfs.Chmod(filesystem, f.resolvePath(path), mode)
}

func (f *FileDescriptors) Stat(path string) (os.FileInfo, error) {
	return hackpadfs.Stat(filesystem, f.resolvePath(path))
}

func (f *FileDescriptors) Lstat(path string) (os.FileInfo, error) {
	return hackpadfs.LstatOrStat(filesystem, f.resolvePath(path))
}

func (f *FileDescriptors) Mkdir(path string, mode os.FileMode) error {
	return hackpadfs.Mkdir(filesystem, f.resolvePath(path), mode)
}

func (f *FileDescriptors) MkdirAll(path string, mode os.FileMode) error {
	return hackpadfs.MkdirAll(filesystem, f.resolvePath(path), mode)
}

func (f *FileDescriptors) Unlink(path string) error {
	path = f.resolvePath(path)
	info, err := hackpadfs.Stat(filesystem, path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.ErrPermission
	}
	return hackpadfs.Remove(filesystem, path)
}

func (f *FileDescriptors) Utimes(path string, atime, mtime time.Time) error {
	return hackpadfs.Chtimes(filesystem, f.resolvePath(path), atime, mtime)
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

func (f *FileDescriptors) Truncate(fd FID, length int64) error {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return interop.BadFileNumber(fd)
	}
	return hackpadfs.TruncateFile(fileDescriptor.file, length)
}

func (f *FileDescriptors) Fsync(fd FID) error {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return interop.BadFileNumber(fd)
	}
	err := hackpadfs.SyncFile(fileDescriptor.file)
	if errors.Is(err, hackpadfs.ErrNotImplemented) {
		err = nil // not all FS implement Sync(), so fall back to a no-op
	}
	return err
}

func (f *FileDescriptors) Rename(oldPath, newPath string) error {
	oldPath = f.resolvePath(oldPath)
	newPath = f.resolvePath(newPath)
	return hackpadfs.Rename(filesystem, oldPath, newPath)
}

func (f *FileDescriptors) Fchmod(fd FID, mode os.FileMode) error {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return interop.BadFileNumber(fd)
	}
	return hackpadfs.Chmod(filesystem, f.resolvePath(fileDescriptor.FileName()), mode)
}

type LockAction int

const (
	LockShared LockAction = iota
	LockExclusive
	Unlock
)

var (
	processFileLocks = make(map[string]*sync.RWMutex)
	newFileLockMu    sync.Mutex
)

func (f *FileDescriptors) Flock(fd FID, action LockAction) error {
	fileDescriptor := f.files[fd]
	if fileDescriptor == nil {
		return interop.BadFileNumber(fd)
	}
	absPath := fileDescriptor.FileName()
	if _, ok := processFileLocks[absPath]; !ok {
		newFileLockMu.Lock()
		if _, ok := processFileLocks[absPath]; !ok {
			processFileLocks[absPath] = new(sync.RWMutex)
		}
		newFileLockMu.Unlock()
	}
	lock := processFileLocks[absPath]
	switch action {
	case LockShared, LockExclusive:
		// TODO support shared locks
		lock.Lock()
	case Unlock:
		lock.Unlock()
	default:
		return interop.ErrNotImplemented
	}
	return nil
}

func (f *FileDescriptors) RawFID(fid FID) (io.Reader, error) {
	if _, ok := f.files[fid]; !ok {
		return nil, interop.BadFileNumber(fid)
	}
	return f.files[fid].file, nil
}

func (f *FileDescriptors) RawFIDs() []io.Reader {
	results := make([]io.Reader, 0, len(f.files))
	for _, f := range f.files {
		results = append(results, f.file)
	}
	return results
}
