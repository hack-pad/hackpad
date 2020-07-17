package storer

import (
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
	path = fsutil.NormalizePath(path)
	if file == nil {
		return f.SetFileRecord(path, nil)
	}
	return f.SetFileRecord(path, &file.FileRecord)
}
