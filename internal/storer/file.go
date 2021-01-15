package storer

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/pkg/errors"
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
	Data     interop.Blob
	DirNames []string
	ModTime  time.Time
	Mode     os.FileMode
}

func (fs *Fs) newFile(path string, flag int, mode os.FileMode) *File {
	return &File{
		flag: flag,
		fileData: &fileData{
			storer: fs.fileStorer,
			path:   path,
			FileRecord: FileRecord{
				Data:    interop.NewBlobBytes(nil),
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
	blob, n, err := f.ReadBlobAt(len(p), off)
	if blob != nil {
		copy(p, blob.Bytes())
	}
	return n, err
}

func (f *File) ReadBlobAt(length int, off int64) (blob interop.Blob, n int, err error) {
	if off >= int64(f.Data.Len()) {
		return nil, 0, io.EOF
	}
	max := int64(f.Data.Len())
	end := off + int64(length)
	if end > max {
		end = max
	}
	blob, err = f.Data.Slice(off, end)
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
		newOffset = int64(f.Data.Len()) + offset
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
	return f.WriteBlobAt(interop.NewBlobBytes(p), off)
}

func (f *File) WriteBlobAt(p interop.Blob, off int64) (n int, err error) {
	if f.flag&syscall.O_APPEND != 0 {
		off = int64(f.fileData.Data.Len())
	}

	endIndex := off + int64(p.Len())
	if int64(f.Data.Len()) < endIndex {
		err := f.Data.Grow(endIndex - int64(f.Data.Len()))
		if err != nil {
			return 0, err
		}
	}
	n, err = f.Data.Set(p, off)
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
	return FileInfo{*f.fileData}, nil
}

func (f *File) Sync() error {
	f.updateModTime()
	return f.save()
}

func (f *File) Truncate(size int64) error {
	length := int64(f.Data.Len())
	switch {
	case size < 0:
		return os.ErrInvalid
	case size == length:
		return nil
	case size > length:
		err := f.Data.Grow(size - length)
		if err != nil {
			return err
		}
	case size < length:
		data := f.Data.Bytes()
		data = data[:size]
		f.Data = interop.NewBlobBytes(data)
	}
	f.updateModTime()
	return f.save()
}

func (f *File) WriteString(s string) (ret int, err error) {
	return f.Write([]byte(s))
}
