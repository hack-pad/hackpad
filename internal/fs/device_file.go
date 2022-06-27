package fs

import (
	"io"

	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

type deviceFile struct {
	name      string
	rawDevice io.ReadWriteCloser
}

var _ hackpadfs.File = &deviceFile{}

func newDeviceFile(name string, rawDevice io.ReadWriteCloser) *deviceFile {
	return &deviceFile{
		name:      name,
		rawDevice: rawDevice,
	}
}

func (d *deviceFile) Read(p []byte) (n int, err error) {
	n, err = d.rawDevice.Read(p)
	return n, errors.WithStack(err)
}

func (d *deviceFile) Write(p []byte) (n int, err error) {
	n, err = d.rawDevice.Write(p)
	return n, errors.WithStack(err)
}

func (d *deviceFile) Close() error {
	return d.rawDevice.Close()
}

func (d *deviceFile) Stat() (hackpadfs.FileInfo, error) {
	return newNamedFileInfo(d.name), nil
}
