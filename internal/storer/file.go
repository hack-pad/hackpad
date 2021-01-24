package storer

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

var (
	_ blob.Reader   = &File{}
	_ blob.ReaderAt = &File{}
	_ blob.Writer   = &File{}
	_ blob.WriterAt = &File{}
)

type File struct {
	*fileData
	offset   int64
	flag     int
	dirCount int
}

type fileData struct {
	FileRecord

	path   string // path is stored as the "key", keeping it here is for generating os.FileInfo's
	storer *fileStorer
}

type FileRecord struct {
	dataOnce sync.Once
	dataDone atomic.Bool
	data     blob.Blob
	DataFn   func() (blob.Blob, error)

	DirNames    []string
	InitialSize int64 // fallback size, enables lazy-loaded Data
	ModTime     time.Time
	Mode        os.FileMode
}

func (f *FileRecord) Data() blob.Blob {
	var err error
	f.dataOnce.Do(func() {
		f.data, err = f.DataFn()
	})
	if err != nil {
		panic(err) // data fn should never fail. IDB data will only fail if types are wrong
	}
	f.dataDone.Store(true)
	return f.data
}

func (f *FileRecord) Size() int64 {
	if f.dataDone.Load() {
		return int64(f.data.Len())
	}
	return f.InitialSize
}

func (fs *Fs) newFile(path string, flag int, mode os.FileMode) *File {
	return &File{
		flag: flag,
		fileData: &fileData{
			storer: fs.fileStorer,
			path:   path,
			FileRecord: FileRecord{
				DataFn: func() (blob.Blob, error) {
					return blob.NewFromBytes(nil), nil
				},
				ModTime: time.Now(),
				Mode:    mode,
			},
		},
	}
}

func (f *fileData) save() error {
	return f.storer.SetFile(f.path, f)
}

func (f *fileData) info() os.FileInfo {
	return &FileInfo{Record: &f.FileRecord, Path: f.path}
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

func (f *File) ReadBlob(length int) (blob blob.Blob, n int, err error) {
	blob, n, err = f.ReadBlobAt(length, f.offset)
	f.offset += int64(n)
	return
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	blob, n, err := f.ReadBlobAt(len(p), off)
	if blob != nil {
		copy(p, blob.Bytes())
	}
	return n, err
}

func (f *File) ReadBlobAt(length int, off int64) (blob blob.Blob, n int, err error) {
	if off >= int64(f.Size()) {
		return nil, 0, io.EOF
	}
	max := int64(f.Size())
	end := off + int64(length)
	if end > max {
		end = max
	}
	blob, err = f.Data().Slice(off, end)
	if err != nil {
		return nil, 0, err
	}
	n = blob.Len()
	if off+int64(n) == max {
		return blob, n, io.EOF
	}
	return blob, n, nil
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	newOffset := f.offset
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset += offset
	case io.SeekEnd:
		newOffset = int64(f.Size()) + offset
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

func (f *File) WriteBlob(p blob.Blob) (n int, err error) {
	n, err = f.WriteBlobAt(p, f.offset)
	f.offset += int64(n)
	return
}

func (f *File) WriteAt(p []byte, off int64) (n int, err error) {
	return f.WriteBlobAt(blob.NewFromBytes(p), off)
}

func (f *File) WriteBlobAt(p blob.Blob, off int64) (n int, err error) {
	if f.flag&syscall.O_APPEND != 0 {
		off = int64(f.fileData.Size())
	}

	endIndex := off + int64(p.Len())
	if int64(f.Size()) < endIndex {
		err := f.Data().Grow(endIndex - int64(f.Size()))
		if err != nil {
			return 0, err
		}
	}
	n, err = f.Data().Set(p, off)
	if err != nil {
		return n, err
	}
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
	f.dirCount = endCount
	if count > 0 && f.dirCount == len(f.DirNames) {
		return names, io.EOF
	}
	return names, nil
}

func (f *File) Stat() (os.FileInfo, error) {
	return FileInfo{Record: &f.FileRecord, Path: f.path}, nil
}

func (f *File) Sync() error {
	f.updateModTime()
	return f.save()
}

func (f *File) Truncate(size int64) error {
	length := int64(f.Size())
	switch {
	case size < 0:
		return os.ErrInvalid
	case size == length:
		return nil
	case size > length:
		err := f.Data().Grow(size - length)
		if err != nil {
			return err
		}
	case size < length:
		f.Data().Truncate(size)
	}
	f.updateModTime()
	return f.save()
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}
