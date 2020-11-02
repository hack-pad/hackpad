package storer

import (
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"time"

	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
)

type File struct {
	*fileData
	offset   int64
	dirCount int
}

type fileData struct {
	FileRecord

	path   string // path is stored as the "key", keeping it here is for generating os.FileInfo's
	storer *fileStorer
}

type FileRecord struct {
	Data     []byte
	DirNames []string
	ModTime  time.Time
	Mode     os.FileMode
}

func (fs *Fs) newFile(path string, mode os.FileMode) *File {
	return &File{
		fileData: &fileData{
			storer: fs.fileStorer,
			path:   path,
			FileRecord: FileRecord{
				ModTime: time.Now(),
				Mode:    mode,
			},
		},
	}
}

func (f *fileData) save() error {
	return f.storer.SetFile(f.path, f)
}

func (f fileData) info() os.FileInfo {
	return &FileInfo{f}
}

func (f *File) Close() error {
	if f.fileData == nil {
		return os.ErrClosed
	}
	f.updateModTime()
	f.fileData = nil
	return nil
}

func (f *File) updateModTime() {
	f.ModTime = time.Now()
}

func (f *File) Read(p []byte) (n int, err error) {
	n, err = f.ReadAt(p, f.offset)
	f.offset += int64(n)
	return
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(f.Data)) {
		return 0, io.EOF
	}
	log.Warn("Read ", f.path, " ", off, "->", len(p), " / ", len(f.Data), "\n"+string(debug.Stack()))
	max := int64(len(f.Data))
	end := off + int64(len(p))
	if end > max {
		end = max
	}
	n = copy(p, f.Data[off:end])
	if off+int64(n) == max {
		return n, io.EOF
	}
	return n, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	newOffset := f.offset
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		newOffset = int64(len(f.Data)) + offset
	default:
		return 0, errors.Errorf("Unknown seek type: %d", whence)
	}
	if newOffset < 0 {
		return 0, errors.Errorf("Cannot seek to negative offset: %d", newOffset)
	}
	f.offset = newOffset
	return newOffset, nil
}

func (f *File) Write(p []byte) (n int, err error) {
	n, err = f.WriteAt(p, f.offset)
	f.offset += int64(n)
	return
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	endIndex := off + int64(len(p))
	log.Warn("Write ", endIndex)
	if int64(len(f.Data)) < endIndex {
		f.Data = append(f.Data, make([]byte, endIndex-int64(len(f.Data)))...)
	}
	n = copy(f.Data[off:], p)
	if n != 0 {
		f.updateModTime()
	}
	err = f.fileData.save()
	return
}

func (f *File) Name() string {
	return f.path
}

func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	names, err := f.Readdirnames(count)
	if err != nil {
		return nil, err
	}

	var infos []os.FileInfo
	for _, name := range names {
		path := filepath.Join(f.path, name)
		info, err := f.storer.fs.Stat(path)
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (f *File) Readdirnames(count int) ([]string, error) {
	log.Print("reading dir: ", f.Name(), " ", f.dirCount, " ", f.DirNames)
	if count > 0 && f.dirCount == len(f.DirNames) {
		return nil, io.EOF
	}

	endCount := f.dirCount + count
	if count <= 0 || endCount > len(f.DirNames) {
		endCount = len(f.DirNames)
	}

	// create sorted copy of dir child names
	allNames := make([]string, len(f.DirNames))
	copy(allNames, f.DirNames)
	sort.Strings(allNames)

	names := allNames[f.dirCount:endCount]
	log.Print("finished reading dir:", f.Name(), "; names: ", names)
	f.dirCount = endCount
	if count > 0 && f.dirCount == len(f.DirNames) {
		return names, io.EOF
	}
	return names, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return FileInfo{*f.fileData}, nil
}

func (f *File) Sync() error {
	f.updateModTime()
	return f.save()
}

func (f *File) Truncate(size int64) error {
	length := int64(len(f.Data))
	switch {
	case size < 0:
		return os.ErrInvalid
	case size == length:
		return nil
	case size > length:
		f.Data = append(f.Data, make([]byte, size-length)...)
	case size < length:
		f.Data = f.Data[:size]
	}
	f.updateModTime()
	return f.save()
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}
