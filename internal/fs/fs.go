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

func Mounts() (pathsToFSName map[string]string) {
	return filesystem.Mounts()
}

func OverlayStorage(mountPath string, s storer.Storer) error {
	fs := storer.New(s)
	return filesystem.Mount(mountPath, fs)
}

func OverlayZip(mountPath string, z *zip.Reader) error {
	return filesystem.Mount(mountPath, zipfs.New(z))
}

func OverlayTarGzip(mountPath string, r io.Reader) error {
	fs, err := tarfs.New(r, afero.NewMemMapFs())
	if err != nil {
		return err
	}
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
