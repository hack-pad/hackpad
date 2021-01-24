package storer

import (
	"os"
	"path/filepath"
	"time"
)

type FileInfo struct {
	Record *FileRecord
	Path   string
}

func (f FileInfo) Name() string {
	return filepath.Base(f.Path)
}

func (f FileInfo) Size() int64 {
	return f.Record.Size()
}

func (f FileInfo) Mode() os.FileMode {
	return f.Record.Mode
}

func (f FileInfo) ModTime() time.Time {
	return f.Record.ModTime
}

func (f FileInfo) IsDir() bool {
	return f.Record.Mode.IsDir()
}

func (f FileInfo) Sys() interface{} {
	return nil
}
