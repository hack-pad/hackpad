package mountfs

import (
	"os"
	"strings"
	"time"

	"github.com/spf13/afero"
)

type mountedFs struct {
	mount
}

func (m mountedFs) mountPath(path string) (s string) {
	return strings.TrimPrefix(path, m.path)
}

func (m mountedFs) Name() string {
	return m.mount.fs.Name()
}

func (m mountedFs) Create(name string) (afero.File, error) {
	return m.mount.fs.Create(m.mountPath(name))
}

func (m mountedFs) Mkdir(name string, perm os.FileMode) error {
	return m.mount.fs.Mkdir(m.mountPath(name), perm)
}

func (m mountedFs) MkdirAll(path string, perm os.FileMode) error {
	return m.mount.fs.MkdirAll(m.mountPath(path), perm)
}

func (m mountedFs) Open(name string) (afero.File, error) {
	return m.mount.fs.Open(m.mountPath(name))
}

func (m mountedFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return m.mount.fs.OpenFile(m.mountPath(name), flag, perm)
}

func (m mountedFs) Remove(name string) error {
	return m.mount.fs.Remove(m.mountPath(name))
}

func (m mountedFs) RemoveAll(path string) error {
	return m.mount.fs.RemoveAll(m.mountPath(path))
}

func (m mountedFs) Rename(oldname, newname string) error {
	return m.mount.fs.Rename(m.mountPath(oldname), m.mountPath(newname))
}

func (m mountedFs) Stat(name string) (os.FileInfo, error) {
	return m.mount.fs.Stat(m.mountPath(name))
}

func (m mountedFs) Chmod(name string, mode os.FileMode) error {
	return m.mount.fs.Chmod(m.mountPath(name), mode)
}

func (m mountedFs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return m.mount.fs.Chtimes(m.mountPath(name), atime, mtime)
}

func (m mountedFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fs := m.mount.fs
	name = m.mountPath(name)
	if lstater, ok := fs.(afero.Lstater); ok {
		return lstater.LstatIfPossible(name)
	}
	info, err := fs.Stat(name)
	return info, false, err
}
