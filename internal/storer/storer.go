package storer

import (
	"encoding/json"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/spf13/afero"
)

// Storer holds arbitrary data at the given 'key' locations. This can be mapped to a key-value store fairly easily.
type Storer interface {
	// GetData retrieves data for the given 'key'. Returns an error if the key is not set or could not be retrieved.
	// If the key is not set, the error must satisfy os.IsNotExist().
	GetData(key string) ([]byte, error)
	// SetData assigns 'data' to the given 'key'. Returns an error if the data could not be set.
	SetData(key string, data []byte) error
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
	data, err := f.GetData(path)
	if err != nil {
		return nil, err
	}
	var file fileData
	err = json.Unmarshal(data, &file)
	file.path = path
	file.storer = f
	return &File{fileData: &file}, err
}

// SetFile write the 'file' data to the store at 'path'. If 'file' is nil, the file is deleted.
func (f *fileStorer) SetFile(path string, file *fileData) error {
	path = fsutil.NormalizePath(path)
	if file == nil {
		return f.SetData(path, nil)
	}
	data, err := json.Marshal(file)
	if err == nil {
		err = f.SetData(path, data)
	}
	return err
}
