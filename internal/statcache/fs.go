package statcache

import (
	"sync"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
	"github.com/pkg/errors"
)

type FS struct {
	*indexeddb.FS
	shouldCache     func(path string) bool
	infoCache       sync.Map
	dirEntriesCache sync.Map
}

func NewFS(fs *indexeddb.FS, shouldCache func(path string) bool) *FS {
	return &FS{
		FS:          fs,
		shouldCache: shouldCache,
	}
}

func (fs *FS) Open(name string) (hackpadfs.File, error) {
	return fs.OpenFile(name, hackpadfs.FlagReadOnly, 0)
}

func (fs *FS) OpenFile(name string, flag int, mode hackpadfs.FileMode) (_ hackpadfs.File, err error) {
	defer func() {
		if v := recover(); v != nil {
			err = errors.Errorf("%v", v)
		}
	}()

	if flag&hackpadfs.FlagCreate != 0 {
		fs.dirEntriesCache.Delete(name)
	}
	f, err := fs.FS.OpenFile(name, flag, mode)
	if err == nil && flag&(hackpadfs.FlagReadWrite|hackpadfs.FlagWriteOnly) != 0 {
		f = newFile(name, fs, f.(keyvalueFile))
	}
	return f, err
}

func (fs *FS) Stat(name string) (hackpadfs.FileInfo, error) {
	if !fs.shouldCache(name) {
		return fs.FS.Stat(name)
	}
	infoInterface, ok := fs.infoCache.Load(name)
	if ok {
		return infoInterface.(hackpadfs.FileInfo), nil
	}

	info, err := fs.FS.Stat(name)
	if err == nil {
		fs.infoCache.Store(name, info)
	}
	return info, err
}

func (fs *FS) Rename(oldname, newname string) error {
	fs.infoCache.Delete(oldname)
	fs.infoCache.Delete(newname)
	fs.dirEntriesCache.Delete(oldname)
	fs.dirEntriesCache.Delete(newname)
	return fs.FS.Rename(oldname, newname)
}

func (fs *FS) Remove(name string) error {
	fs.infoCache.Delete(name)
	fs.dirEntriesCache.Delete(name)
	return fs.FS.Remove(name)
}

func (fs *FS) ReadDir(name string) ([]hackpadfs.DirEntry, error) {
	if !fs.shouldCache(name) {
		return hackpadfs.ReadDir(fs.FS, name)
	}

	dirEntriesInterface, ok := fs.dirEntriesCache.Load(name)
	if ok {
		return dirEntriesInterface.([]hackpadfs.DirEntry), nil
	}

	dirEntries, err := hackpadfs.ReadDir(fs.FS, name)
	if err == nil {
		fs.dirEntriesCache.Store(name, dirEntries)
	}
	return dirEntries, err
}
