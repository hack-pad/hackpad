package storer

import (
	"fmt"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/spf13/afero"
)

// Storer holds arbitrary data at the given 'key' locations. This can be mapped to a key-value store fairly easily.
type Storer interface {
	// GetFileRecord retrieves a file for the given 'path' and stores it in 'dest'.
	// Returns an error if the path was not found or could not be retrieved.
	// If the path was not found, the error must satisfy os.IsNotExist().
	GetFileRecord(path string, dest *FileRecord) error
	// SetFileRecord assigns 'data' to the given 'path'. Returns an error if the data could not be set.
	SetFileRecord(path string, src *FileRecord) error
}

type BatchGetter interface {
	GetFileRecords(paths []string, dest []*FileRecord) []error
}

type fileStorer struct {
	Storer
	fs afero.Fs
}

func newFileStorer(s Storer, sourceFS afero.Fs) *fileStorer {
	return &fileStorer{Storer: s, fs: sourceFS}
}

// GetFile returns a file for 'path' if it exists, os.ErrNotExist otherwise
func (f *fileStorer) GetFile(path string) (*File, error) {
	path = fsutil.NormalizePath(path)
	file := fileData{
		path:   path,
		storer: f,
	}
	err := f.GetFileRecord(path, &file.FileRecord)
	return &File{fileData: &file}, err
}

// SetFile write the 'file' data to the store at 'path'. If 'file' is nil, the file is deleted.
func (f *fileStorer) SetFile(path string, file *fileData) error {
	if file == nil {
		return <-QueueSetFileRecord(f.Storer, path, nil)
	}
	return <-QueueSetFileRecord(f.Storer, path, &file.FileRecord)
}

func (f *fileStorer) GetFiles(paths ...string) ([]*File, []error) {
	fileRecords := make([]*FileRecord, len(paths))
	files := make([]*File, len(paths))
	for i := range files {
		path := fsutil.NormalizePath(paths[i])
		files[i] = &File{
			fileData: &fileData{
				path:   path,
				storer: f,
			},
		}
		fileRecords[i] = &files[i].FileRecord
	}
	errs := GetFileRecords(f.Storer, paths, fileRecords)
	return files, errs
}

func GetFileRecords(s Storer, paths []string, dest []*FileRecord) []error {
	if len(paths) != len(dest) {
		panic(fmt.Sprintf("GetFileRecords: Paths and dest lengths must be equal: %d != %d", len(paths), len(dest)))
	}
	if batcher, ok := s.(BatchGetter); ok {
		return batcher.GetFileRecords(paths, dest)
	}

	errs := make([]error, len(paths))
	for i := range paths {
		path, destI := paths[i], dest[i]
		errs[i] = s.GetFileRecord(path, destI)
	}
	return errs
}
