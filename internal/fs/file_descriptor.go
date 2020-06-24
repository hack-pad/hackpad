package fs

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/log"
	"github.com/spf13/afero"
	"go.uber.org/atomic"
)

type FID = common.FID

type fileDescriptor struct {
	id FID
	*fileCore
}

type fileCore struct {
	file afero.File
	mode os.FileMode

	openMu     sync.Mutex
	openCounts map[common.PID]*atomic.Uint64
}

func NewFileDescriptor(fid FID, absPath string, flags int, mode os.FileMode) (*fileDescriptor, error) {
	file, err := getFile(absPath, flags, mode)
	descriptor := newIrregularFileDescriptor(fid, file, mode)
	return descriptor, err
}

func newIrregularFileDescriptor(fid FID, file afero.File, mode os.FileMode) *fileDescriptor {
	return &fileDescriptor{
		id: fid,
		fileCore: &fileCore{
			file:       file,
			mode:       mode,
			openCounts: make(map[common.PID]*atomic.Uint64),
		},
	}
}

func (fd *fileDescriptor) Dup(fid FID) *fileDescriptor {
	fdCopy := *fd
	fdCopy.id = fid
	return &fdCopy
}

func (fd *fileDescriptor) FileName() string {
	return fd.file.Name()
}

func (fd *fileDescriptor) String() string {
	return fmt.Sprintf("%15s [%d] open=%v", fd.file.Name(), fd.id, openCountToString(fd.openCounts))
}

func (fd *fileDescriptor) Open(pid common.PID) {
	count, ok := fd.openCounts[pid]
	if ok {
		count.Inc()
		return
	}
	fd.fileCore.openMu.Lock()
	if count, ok := fd.openCounts[pid]; ok {
		count.Inc()
	} else {
		fd.openCounts[pid] = atomic.NewUint64(1)
	}
	fd.fileCore.openMu.Unlock()
}

// Close decrements this process's open count. If the open count is 0, then it locks and runs cleanup.
// If the open count is zero for all processes, then the internal file is closed.
func (fd *fileDescriptor) Close(pid common.PID, locker sync.Locker, cleanUpFile func()) error {
	count := fd.openCounts[pid]
	if count == nil || count.Load() <= 0 {
		return nil
	}

	if count.Dec() > 0 {
		return nil
	}
	// if this process's open count is 0, then use 'locker' and 'cleanUpFile' to remove it from the parent
	locker.Lock()
	fd.openMu.Lock()
	cleanedUp, err := fd.unsafeClose(pid)
	if cleanedUp {
		cleanUpFile()
	}
	fd.openMu.Unlock()
	locker.Unlock()
	return err
}

func (fd *fileDescriptor) unsafeClose(pid common.PID) (cleanUpFile bool, err error) {
	count, ok := fd.openCounts[pid]
	if !ok {
		return
	}
	if count.Load() == 0 {
		delete(fd.openCounts, pid)
		cleanUpFile = true
	}

	if len(fd.openCounts) == 0 {
		// if this fd is closed everywhere, then close the file
		err = fd.file.Close()
	}
	return
}

func openCountToString(openCounts map[common.PID]*atomic.Uint64) string {
	var s strings.Builder
	s.WriteString("{")
	for pid, count := range openCounts {
		s.WriteString(fmt.Sprintf(" %d:%d", pid, count.Load()))
	}
	s.WriteString(" }")
	return s.String()
}

func (fd *fileDescriptor) closeAll(pid common.PID) error {
	fd.openMu.Lock()
	defer fd.openMu.Unlock()

	count := fd.openCounts[pid]
	if count == nil {
		return nil
	}
	var firstErr error
	for count.Load() > 0 {
		count.Dec()
		_, err := fd.unsafeClose(pid)
		if firstErr == nil && err != nil {
			log.Errorf("Failed to close file for PID %d %q: %s", pid, fd.FileName(), err.Error())
			firstErr = err
		}
	}
	return firstErr
}
