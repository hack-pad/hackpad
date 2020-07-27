package fs

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/log"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// readOnlyCacheFs attempts to fix some strange behavior of afero.CacheOnReadFs
type readOnlyCacheFs struct {
	base, cache afero.Fs
}

func newReadOnlyCacheFs(base, cacheLayer afero.Fs) afero.Fs {
	return &readOnlyCacheFs{base: base, cache: cacheLayer}
}

func (fs *readOnlyCacheFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	// only read only allowed (0)
	if flag != os.O_RDONLY {
		return nil, syscall.EPERM
	}

	// if in cache, return that
	_, err := fs.cache.Stat(name)
	if err == nil {
		f, err := fs.cache.OpenFile(name, flag, perm)
		if err != nil {
			log.Warn("Failed cache open for ", name, " ", err.Error())
		}
		return fs.wrapFile(f), err
	}

	// if not in base, fail
	info, err := fs.base.Stat(name)
	if os.IsNotExist(err) {
		log.Warn("Failed base stat for ", name, " ", err.Error())
		return nil, err
	}

	// ensure parent directories exist with same permissions
	if err := fs.copyParentDirs(name); err != nil {
		log.Warn("Failed parent copydir for ", name, " ", err.Error())
		return nil, err
	}

	// copy file to cache
	cacheFile, err := fs.cache.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		log.Warn("Failed cache (copy) open for ", name, " with perm ", info.Mode(), ": ", err.Error())
		return nil, err
	}
	sourceFile, err := fs.base.Open(name)
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(cacheFile, sourceFile); err != nil {
		return nil, err
	}

	f, err := fs.cache.Open(name)
	return fs.wrapFile(f), err
}

func (fs *readOnlyCacheFs) wrapFile(f afero.File) afero.File {
	return &readOnlyCacheFile{
		File: f,
		base: fs.base,
	}
}

func (fs *readOnlyCacheFs) copyParentDirs(name string) error {
	type dirAndMode struct {
		dir  string
		mode os.FileMode
	}
	var dirs []dirAndMode
	for path := fsutil.NormalizePath(name); path != afero.FilePathSeparator; path = filepath.Dir(path) {
		_, err := fs.cache.Stat(path)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return err
		}
		info, err := fs.base.Stat(path)
		if err != nil {
			return errors.Wrap(err, "missing intermediate dir in base "+path)
		} else if info.IsDir() {
			dirs = append(dirs, dirAndMode{path, info.Mode()})
		}
	}
	// create missing dirs in reverse
	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		log.Warn("ROCache: making dir ", dir)
		err := fs.cache.Mkdir(dir.dir, dir.mode)
		if err != nil {
			log.Warn("Failed to make dir ", dir, " ", err.Error())
			return err
		}
	}
	return nil
}

func (fs *readOnlyCacheFs) Stat(name string) (os.FileInfo, error) {
	info, err := fs.base.Stat(name)
	if err != nil {
		log.Warn("Failed to stat base file ", name, ": ", err.Error())
	}
	return info, err
}

func (fs *readOnlyCacheFs) Name() string {
	return "*readOnlyCacheFs"
}

func (fs *readOnlyCacheFs) Create(name string) (afero.File, error)       { return nil, syscall.EPERM }
func (fs *readOnlyCacheFs) Mkdir(name string, perm os.FileMode) error    { return syscall.EPERM }
func (fs *readOnlyCacheFs) MkdirAll(path string, perm os.FileMode) error { return syscall.EPERM }
func (fs *readOnlyCacheFs) Open(name string) (afero.File, error)         { return fs.OpenFile(name, 0, 0) }
func (fs *readOnlyCacheFs) Remove(name string) error                     { return syscall.EPERM }
func (fs *readOnlyCacheFs) RemoveAll(path string) error                  { return syscall.EPERM }
func (fs *readOnlyCacheFs) Rename(oldname, newname string) error         { return syscall.EPERM }
func (fs *readOnlyCacheFs) Chmod(name string, mode os.FileMode) error    { return syscall.EPERM }
func (fs *readOnlyCacheFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return syscall.EPERM
}

type readOnlyCacheFile struct {
	afero.File

	base afero.Fs
}

func (f *readOnlyCacheFile) Readdir(count int) ([]os.FileInfo, error) {
	if count > 0 {
		panic("should never read with specific quantity")
	}
	baseF, err := f.base.Open(f.Name())
	if err != nil {
		log.Warn("failed to open base dir ", err.Error())
		return nil, err
	}
	defer baseF.Close()
	infos, err := baseF.Readdir(count)
	if err != nil {
		log.Warn("failed to read base dir ", err.Error())
	}
	return infos, err
}

func (f *readOnlyCacheFile) Readdirnames(n int) ([]string, error) {
	if n > 0 {
		panic("should never read with specific quantity")
	}
	files, err := afero.ReadDir(f.base, f.Name())
	if err != nil {
		return nil, err
	}
	var names []string
	for _, info := range files {
		names = append(names, filepath.Base(info.Name()))
	}
	return names, nil
}

func (f *readOnlyCacheFile) Stat() (os.FileInfo, error) {
	info, err := f.base.Stat(f.Name())
	if err != nil {
		log.Warn("Failed to stat base file ", f.Name(), ": ", err.Error())
	}
	return info, err
}

func (f *readOnlyCacheFile) Write(p []byte) (n int, err error)              { return 0, syscall.EPERM }
func (f *readOnlyCacheFile) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EPERM }
func (f *readOnlyCacheFile) Truncate(size int64) error                      { return syscall.EPERM }
func (f *readOnlyCacheFile) WriteString(s string) (ret int, err error)      { return 0, syscall.EPERM }
