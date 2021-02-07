package storer

import (
	"fmt"

	"github.com/johnstarich/go-wasm/internal/fsutil"
)

type BatchGetter interface {
	GetFileRecords(paths []string, dest []*FileRecord) []error
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
