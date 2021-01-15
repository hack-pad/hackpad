package storer

import (
	"os"
	"path/filepath"
	"time"
)

type FileInfo struct {
	fileData
}

func (f FileInfo) Name() string {
	return filepath.Base(f.path)
}

func (f FileInfo) Size() int64 {
	return int64(f.Data.Len())
}

func (f FileInfo) Mode() os.FileMode {
	return f.fileData.Mode
}

func (f FileInfo) ModTime() time.Time {
	return f.fileData.ModTime
}

func (f FileInfo) IsDir() bool {
	return f.fileData.Mode&os.ModeDir == os.ModeDir
}

func (f FileInfo) Sys() interface{} {
	return nil
}
