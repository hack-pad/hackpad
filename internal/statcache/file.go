package statcache

import (
	"fmt"
	"io"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

// keyvalueFile is a copy of the interface *keyvalue.file implements, to expose the same API in a wrapped struct
type keyvalueFile interface {
	hackpadfs.File
	hackpadfs.SeekerFile
	hackpadfs.TruncaterFile
}

type keyvalueFileReader interface {
	keyvalueFile
	hackpadfs.DirReaderFile
	io.ReaderAt
	blob.Reader
	blob.ReaderAt
}

type keyvalueFileWriter interface {
	keyvalueFile
	io.WriterAt
	blob.Writer
	blob.WriterAt
	hackpadfs.ReadWriterFile
}

type keyvalueFileReadWriter interface {
	keyvalueFileReader
	keyvalueFileWriter
}

type fileReader struct {
	keyvalueFileReader

	path string
	fs   *FS
}

type fileWriter struct {
	keyvalueFileWriter

	path string
	fs   *FS
}

type fileReadWriter struct {
	keyvalueFileReadWriter

	path string
	fs   *FS
}

func newFile(path string, fs *FS, f keyvalueFile) keyvalueFile {
	switch f := f.(type) {
	case keyvalueFileReadWriter:
		return &fileReadWriter{keyvalueFileReadWriter: f, path: path, fs: fs}
	case keyvalueFileWriter:
		return &fileWriter{keyvalueFileWriter: f, path: path, fs: fs}
	case keyvalueFileReader:
		return &fileReader{keyvalueFileReader: f, path: path, fs: fs}
	default:
		panic(fmt.Sprintf("Unknown type: %T", f))
	}
}

func readDir(fs *FS, path string, file keyvalueFile, n int) ([]hackpadfs.DirEntry, error) {
	if n > 0 {
		return hackpadfs.ReadDirFile(file, n)
	}
	return fs.ReadDir(path)
}

func (f *fileReadWriter) Stat() (hackpadfs.FileInfo, error) { return f.fs.Stat(f.path) }
func (f *fileReadWriter) ReadDir(n int) ([]hackpadfs.DirEntry, error) {
	return readDir(f.fs, f.path, f, n)
}
func (f *fileReadWriter) Close() error {
	f.fs.infoCache.Delete(f.path)
	return f.keyvalueFileReadWriter.Close()
}

func (f *fileReader) Stat() (hackpadfs.FileInfo, error)           { return f.fs.Stat(f.path) }
func (f *fileReader) ReadDir(n int) ([]hackpadfs.DirEntry, error) { return readDir(f.fs, f.path, f, n) }
func (f *fileReader) Close() error {
	f.fs.infoCache.Delete(f.path)
	return f.keyvalueFileReader.Close()
}

func (f *fileWriter) Stat() (hackpadfs.FileInfo, error) { return f.fs.Stat(f.path) }
func (f *fileWriter) Close() error {
	f.fs.infoCache.Delete(f.path)
	return f.keyvalueFileWriter.Close()
}
