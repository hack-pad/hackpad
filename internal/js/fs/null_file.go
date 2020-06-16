package fs

import (
	"io"
	"os"
	"time"

	"github.com/spf13/afero"
)

type nullFile struct{}

func newNullFile() afero.File {
	return nullFile{}
}

func (f nullFile) Close() error                                   { return nil }
func (f nullFile) Read(p []byte) (n int, err error)               { return len(p), io.EOF }
func (f nullFile) ReadAt(p []byte, off int64) (n int, err error)  { return len(p), io.EOF }
func (f nullFile) Seek(offset int64, whence int) (int64, error)   { return 0, io.EOF }
func (f nullFile) Write(p []byte) (n int, err error)              { return len(p), nil }
func (f nullFile) WriteAt(p []byte, off int64) (n int, err error) { return len(p), nil }
func (f nullFile) Name() string                                   { return "null" }
func (f nullFile) Readdir(count int) ([]os.FileInfo, error)       { return nil, nil }
func (f nullFile) Readdirnames(n int) ([]string, error)           { return nil, nil }
func (f nullFile) Stat() (os.FileInfo, error)                     { return nullStat{}, nil }
func (f nullFile) Sync() error                                    { return nil }
func (f nullFile) Truncate(size int64) error                      { return nil }
func (f nullFile) WriteString(s string) (ret int, err error)      { return len(s), nil }

type nullStat struct{}

func (s nullStat) Name() string       { return "null" }
func (s nullStat) Size() int64        { return 0 }
func (s nullStat) Mode() os.FileMode  { return 0 }
func (s nullStat) ModTime() time.Time { return time.Time{} }
func (s nullStat) IsDir() bool        { return false }
func (s nullStat) Sys() interface{}   { return nil }
