// +build js

package fs

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/nativeio"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/spf13/afero"
)

type NativeIOFs struct {
	afero.Fs
	storer   *nioStorer
	basePath string
}

func NewNativeIOFs(path string) (*NativeIOFs, error) {
	manager, err := nativeio.New()
	store := &nioStorer{
		manager: manager,
	}
	fs := storer.New(store)
	basePathFs := afero.NewBasePathFs(fs, path)
	return &NativeIOFs{
		Fs:       basePathFs,
		storer:   store,
		basePath: path,
	}, err
}

func (fs *NativeIOFs) Clear() error {
	names, err := fs.storer.manager.GetAll()
	if err != nil {
		return err
	}
	for _, n := range names {
		err := fs.storer.manager.Remove(n)
		if err != nil {
			return err
		}
	}
	return nil
}

type nioStorer struct {
	manager *nativeio.Manager
}

func (n *nioStorer) GetFileRecord(path string, dest *storer.FileRecord) error {
	path = fsutil.NormalizePath(path)
	name := pathToName(path)
	f, isDir, err := n.open(name)
	if err != nil {
		return err
	}
	defer f.Close()

	size, err := f.Size()
	if err != nil {
		return err
	}
	dest.InitialSize = int64(size)
	dest.ModTime = time.Unix(0, 0) // TODO not supported by nativeio
	dest.Mode = 0700               // TODO not supported by nativeio
	if isDir {
		dest.Mode |= os.ModeDir
		dest.DirNamesFn = func() ([]string, error) {
			names, err := n.getDirNames(path)
			return names, err
		}
		dest.DataFn = func() (blob.Blob, error) {
			return blob.NewFromBytes(nil), nil
		}
	} else {
		dest.DirNamesFn = func() ([]string, error) {
			return nil, nil
		}
		dest.DataFn = func() (blob.Blob, error) {
			f, _, err := n.open(name)
			if err != nil {
				return nil, err
			}
			defer f.Close()
			size, err := f.Size()
			if err != nil {
				return nil, err
			}
			b, _, err := f.ReadBlobAt(int(size), 0)
			return b, err
		}
	}
	return nil
}

func (n *nioStorer) open(name string) (file *nativeio.File, isDir bool, err error) {
	// TODO This hack implements a more typical "open" behavior, where non-existent files are not created when opened, returning an error instead.
	path, err := nameToPath(name)
	if err != nil {
		return nil, false, err
	}
	dirPath := path
	// Since file mode cannot be stored with nativeIO,
	// implicitly store "is a directory" with a trailing slash.
	if dirPath != afero.FilePathSeparator {
		dirPath += afero.FilePathSeparator
	}
	dirName := pathToName(dirPath)

	fileNames, err := n.manager.GetAll()
	if err != nil {
		return nil, false, err
	}
	for _, fileName := range fileNames {
		switch fileName {
		case dirName:
			file, err = n.manager.OpenOrCreate(dirName)
			return file, true, err
		case name:
			file, err = n.manager.OpenOrCreate(name)
			return file, false, err
		}
	}
	return nil, false, os.ErrNotExist
}

func (n *nioStorer) getDirNames(path string) ([]string, error) {
	// TODO this is extremely inefficient, but currently no built-in way to get child file names
	path = fsutil.NormalizePath(path)
	name := pathToName(path)
	f, isDir, err := n.open(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	if !isDir {
		return nil, afero.ErrNotDir
	}

	fileNames, err := n.manager.GetAll()
	if err != nil {
		return nil, err
	}
	var dirNames []string
	for _, name := range fileNames {
		p, err := nameToPath(name)
		if err != nil {
			return nil, err
		}
		if len(p) > 1 {
			p = strings.TrimSuffix(p, afero.FilePathSeparator)
		}
		dir, base := filepath.Split(p)
		if dir == path {
			dirNames = append(dirNames, base)
		}
	}
	return dirNames, nil
}

func (n *nioStorer) SetFileRecord(path string, src *storer.FileRecord) error {
	path = fsutil.NormalizePath(path)
	if src.Mode.IsDir() && path != afero.FilePathSeparator {
		path += afero.FilePathSeparator
	}
	name := pathToName(path)

	if src == nil {
		return n.manager.Remove(name)
	}

	f, err := n.manager.OpenOrCreate(name)
	if err != nil {
		return err
	}
	defer f.Close()

	if src.Mode.IsDir() {
		return nil
	}
	b := src.Data()
	_, err = f.WriteBlobAt(b, 0)
	if err != nil {
		return err
	}
	size := int64(b.Len())
	err = f.Truncate(uint64(size))
	if err != nil {
		return err
	}
	return f.Flush()
}
