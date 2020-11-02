package fs

import (
	"archive/zip"
	"fmt"
	"io"
	"os"

	"github.com/johnstarich/go-wasm/internal/mountfs"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/internal/tarfs"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
)

var (
	filesystem = mountfs.New(afero.NewMemMapFs())
)

func OverlayStorage(mountPath string, s storer.Storer) error {
	fs := storer.New(s)
	err := filesystem.Mount(mountPath, fs)
	if err != nil {
		return err
	}
	return fs.MkdirAll(mountPath, 0755)
}

func OverlayZip(mountPath string, z *zip.Reader) error {
	return filesystem.Mount(mountPath, zipfs.New(z))
}

func OverlayTarGzip(mountPath string, r io.Reader) error {
	fs := tarfs.New(r)
	return filesystem.Mount(mountPath, fs)
}

// Dump prints out file system statistics
func Dump(basePath string) interface{} {
	var total int64
	err := afero.Walk(filesystem, basePath, func(path string, info os.FileInfo, err error) error {
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
