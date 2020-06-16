package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
)

var (
	ErrNotDir = interop.NewError("not a directory", "ENOTDIR")

	filesystem = afero.NewMemMapFs()
)

func resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(interop.WorkingDirectory(), path)
}

func ReadFile(path string) ([]byte, error) {
	return afero.ReadFile(filesystem, resolvePath(path))
}

func ReadDir(path string) ([]os.FileInfo, error) {
	return afero.ReadDir(filesystem, resolvePath(path))
}

func RemoveDir(path string) error {
	path = resolvePath(path)
	info, err := Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrNotDir
	}
	return filesystem.Remove(path)
}

func Chmod(path string, mode os.FileMode) error {
	return filesystem.Chmod(resolvePath(path), mode)
}

func Stat(path string) (os.FileInfo, error) {
	return filesystem.Stat(resolvePath(path))
}

func Lstat(path string) (os.FileInfo, error) {
	// TODO add proper symlink support
	return filesystem.Stat(resolvePath(path))
}

func Mkdir(path string, mode os.FileMode) error {
	return filesystem.Mkdir(resolvePath(path), mode)
}

func MkdirAll(path string, mode os.FileMode) error {
	return filesystem.MkdirAll(resolvePath(path), mode)
}

func Unlink(path string) error {
	path = resolvePath(path)
	info, err := Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return os.ErrPermission
	}
	return filesystem.Remove(path)
}

func Utimes(path string, atime, mtime time.Time) error {
	return filesystem.Chtimes(resolvePath(path), atime, mtime)
}

func Dump() interface{} {
	var total int64
	err := afero.Walk(filesystem, "/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return err
	}
	return fmt.Sprintf("%d bytes", total)
}
