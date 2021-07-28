package fs

import (
	"io"
	"os"
	"time"

	"github.com/hack-pad/hackpadfs"
)

type nullFile struct {
	name string
}

func newNullFile(name string) hackpadfs.File {
	return nullFile{name: name}
}

func (f nullFile) Close() error                                   { return nil }
func (f nullFile) Read(p []byte) (n int, err error)               { return 0, io.EOF }
func (f nullFile) ReadAt(p []byte, off int64) (n int, err error)  { return 0, io.EOF }
func (f nullFile) Seek(offset int64, whence int) (int64, error)   { return 0, nil }
func (f nullFile) Write(p []byte) (n int, err error)              { return len(p), nil }
func (f nullFile) WriteAt(p []byte, off int64) (n int, err error) { return len(p), nil }
func (f nullFile) Stat() (os.FileInfo, error)                     { return nullStat{f}, nil }
func (f nullFile) Truncate(size int64) error                      { return nil }

type nullStat struct {
	f nullFile
}

func (s nullStat) Name() string       { return s.f.name }
func (s nullStat) Size() int64        { return 0 }
func (s nullStat) Mode() os.FileMode  { return 0 }
func (s nullStat) ModTime() time.Time { return time.Time{} }
func (s nullStat) IsDir() bool        { return false }
func (s nullStat) Sys() interface{}   { return nil }
