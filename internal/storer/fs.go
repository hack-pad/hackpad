package storer

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/johnstarich/go-wasm/internal/rwonly"
	"github.com/spf13/afero"
)

type Fs struct {
	*fileStorer
}

// New returns a file system that relies on data fetched and set on Storer.
// NOTE: 100% NOT thread safe
func New(s Storer) *Fs {
	fs := &Fs{}
	fs.fileStorer = newFileStorer(s, fs)
	return fs
}

func (fs *Fs) wrapperErr(op string, path string, err error) error {
	if err == nil {
		return nil
	}
	if uErr, ok := err.(interface{ Unwrap() error }); ok && uErr != nil {
		err = uErr.Unwrap()
	}
	return &os.PathError{Op: op, Path: path, Err: err}
}

func (fs *Fs) Create(name string) (afero.File, error) {
	file, err := fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	return file, fs.wrapperErr("create", name, err)
}

func (fs *Fs) Mkdir(name string, perm os.FileMode) error {
	_, err := fs.fileStorer.GetFile(name)
	switch {
	case err == nil:
		return &os.PathError{Op: "mkdir", Path: name, Err: os.ErrExist}
	case os.IsNotExist(err):
		file := fs.newDir(name, perm)
		return fs.wrapperErr("mkdir", name, file.save())
	default:
		return &os.PathError{Op: "mkdir", Path: name, Err: err}
	}
}

func (fs *Fs) newDir(name string, perm os.FileMode) *File {
	return fs.newFile(name, 0, os.ModeDir|(perm&os.ModePerm))
}

func (fs *Fs) MkdirAll(path string, perm os.FileMode) error {
	missingDirs, err := fs.findMissingDirs(path)
	if err != nil {
		return err
	}
	for i := len(missingDirs) - 1; i >= 0; i-- { // missingDirs are in reverse order
		name := missingDirs[i]
		file := fs.newDir(name, perm)
		err := file.save()
		err = fs.wrapperErr("mkdirall", name, err)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func statAll(storer Storer, paths []string) ([]os.FileInfo, []error) {
	files := make([]*FileRecord, len(paths))
	for i := range files {
		files[i] = new(FileRecord)
	}

	infos := make([]os.FileInfo, len(files))
	errs := GetFileRecords(storer, paths, files)
	for i := range files {
		infos[i] = FileInfo{Record: files[i], Path: paths[i]}
	}
	return infos, errs
}

// findMissingDirs returns all paths that must be created, in reverse order
func (fs *Fs) findMissingDirs(path string) ([]string, error) {
	path = fsutil.NormalizePath(path)
	var paths []string
	for currentPath := path; currentPath != afero.FilePathSeparator; currentPath = filepath.Dir(currentPath) {
		paths = append(paths, currentPath)
	}
	paths = append(paths, afero.FilePathSeparator)
	infos, errs := statAll(fs.Storer, paths)

	var missingDirs []string
	for i := range paths {
		missing, err := isMissingDir(paths[i], infos[i], errs[i])
		if err != nil {
			return nil, err
		}
		if missing {
			missingDirs = append(missingDirs, paths[i])
		} else {
			return missingDirs, nil
		}
	}
	return missingDirs, nil
}

func isMissingDir(path string, info os.FileInfo, err error) (missing bool, returnedErr error) {
	switch {
	case os.IsNotExist(err):
		return true, nil
	case err != nil:
		return false, err
	case info.IsDir():
		// found a directory in the chain, return early
		return false, nil
	case !info.IsDir():
		// a file is found where we want a directory, fail with ENOTDIR
		return true, &os.PathError{Op: "mkdirall", Path: path, Err: afero.ErrNotDir}
	default:
		return false, nil
	}
}

func (fs *Fs) Open(name string) (afero.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *Fs) OpenFile(name string, flag int, perm os.FileMode) (afFile afero.File, retErr error) {
	paths := []string{name}
	if flag&os.O_CREATE != 0 {
		paths = append(paths, filepath.Dir(name))
	}
	files, errs := fs.fileStorer.GetFiles(paths...)
	storerFile, err := files[0], errs[0]
	switch {
	case err == nil:
		if storerFile.info().IsDir() && flag&os.O_WRONLY != 0 {
			// write-only on a directory isn't allowed on os.OpenFile either
			return nil, &os.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
		}
		storerFile.flag = flag
	case os.IsNotExist(err) && flag&os.O_CREATE != 0:
		// require parent directory
		err := errs[1]
		if err != nil {
			return nil, fs.wrapperErr("stat", name, err)
		}
		storerFile = fs.newFile(name, flag, perm&os.ModePerm)
		if err := storerFile.save(); err != nil {
			return nil, fs.wrapperErr("open", name, err)
		}
	default:
		return nil, fs.wrapperErr("open", name, err)
	}

	var file afero.File = storerFile
	switch {
	case flag&os.O_WRONLY != 0:
		file = rwonly.WriteOnly(file)
	case flag&os.O_RDWR != 0:
	default:
		// os.O_RDONLY = 0
		file = rwonly.ReadOnly(file)
	}

	if flag&os.O_TRUNC != 0 {
		return file, fs.wrapperErr("open", name, file.Truncate(0))
	}
	return file, nil
}

func (fs *Fs) Remove(name string) error {
	file, err := fs.fileStorer.GetFile(name)
	if err != nil {
		return fs.wrapperErr("remove", name, err)
	}

	if file.Mode.IsDir() && len(file.DirNames()) != 0 {
		return &os.PathError{Op: "remove", Path: name, Err: syscall.ENOTEMPTY}
	}
	return fs.fileStorer.SetFile(name, nil)
}

func (fs *Fs) RemoveAll(path string) error {
	return &os.PathError{Op: "removeall", Path: path, Err: syscall.ENOSYS}
}

func (fs *Fs) Rename(oldname, newname string) error {
	oldFile, err := fs.fileStorer.GetFile(oldname)
	if err != nil {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: afero.ErrFileNotFound}
	}
	oldInfo, err := oldFile.Stat()
	if err != nil {
		return err
	}
	if !oldInfo.IsDir() {
		err := fs.fileStorer.SetFile(newname, oldFile.fileData)
		if err != nil {
			return err
		}
		return fs.fileStorer.SetFile(oldname, nil)
	}

	_, err = fs.fileStorer.GetFile(newname)
	if !os.IsNotExist(err) {
		return &os.LinkError{Op: "rename", Old: oldname, New: newname, Err: afero.ErrDestinationExists}
	}

	files, err := oldFile.Readdirnames(-1)
	if err != nil {
		return err
	}
	err = fs.fileStorer.SetFile(newname, oldFile.fileData)
	if err != nil {
		return err
	}
	for _, name := range files {
		err := fs.Rename(filepath.Join(oldname, name), filepath.Join(newname, name))
		if err != nil {
			// TODO don't leave destination in corrupted state (missing file records for dir names)
			return err
		}
	}
	return fs.fileStorer.SetFile(oldname, nil)
}

func (fs *Fs) Stat(name string) (os.FileInfo, error) {
	file, err := fs.fileStorer.GetFile(name)
	if err != nil {
		return nil, fs.wrapperErr("stat", name, err)
	}
	return file.info(), nil
}

func (fs *Fs) Name() string {
	return fmt.Sprintf("storer.Fs(%T)", fs.Storer)
}

func (fs *Fs) Chmod(name string, mode os.FileMode) error {
	file, err := fs.fileStorer.GetFile(name)
	if err != nil {
		return fs.wrapperErr("chmod", name, err)
	}

	const chmodBits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky // Only a subset of bits are allowed to be changed. Documented under os.Chmod()
	file.Mode = (file.Mode & ^chmodBits) | (mode & chmodBits)
	return nil
}

func (fs *Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	file, err := fs.fileStorer.GetFile(name)
	if err != nil {
		return fs.wrapperErr("chtimes", name, err)
	}
	file.ModTime = mtime
	return nil
}
