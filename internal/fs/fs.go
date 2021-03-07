package fs

import (
	"archive/zip"
	"io"
	"os"

	"github.com/johnstarich/go-wasm/internal/mountfs"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/internal/tarfs"
	"github.com/johnstarich/go-wasm/log"
	"github.com/johnstarich/go/datasize"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
)

var (
	filesystem rootFs = mountfs.New(afero.NewMemMapFs())
)

type rootFs interface {
	afero.Fs
	afero.Lstater
	Mounts() map[string]string
	DestroyMount(string) error
	Mount(string, afero.Fs) error
}

func Mounts() (pathsToFSName map[string]string) {
	return filesystem.Mounts()
}

func DestroyMount(path string) error {
	return filesystem.DestroyMount(path)
}

func OverlayStorage(mountPath string, s storer.Storer) error {
	fs, ok := s.(afero.Fs)
	if !ok {
		fs = storer.New(s)
	}
	return filesystem.Mount(mountPath, fs)
}

func OverlayZip(mountPath string, z *zip.Reader) error {
	return filesystem.Mount(mountPath, zipfs.New(z))
}

func OverlayTarGzip(mountPath string, r io.ReadCloser, persist bool) error {
	if !persist {
		underlyingFs := afero.NewMemMapFs()
		fs, err := tarfs.New(r, underlyingFs)
		if err != nil {
			return err
		}
		return filesystem.Mount(mountPath, fs)
	}

	const tarfsDoneMarker = ".tarfs-complete"

	underlyingFs, err := newPersistDB(mountPath)
	if err != nil {
		return err
	}

	_, err = underlyingFs.Stat(tarfsDoneMarker)
	if err == nil {
		// tarfs already completed successfully and is persisted,
		// so close tarfs reader and mount the existing files
		r.Close()
		return filesystem.Mount(mountPath, afero.NewReadOnlyFs(underlyingFs))
	} else {
		// either never untar'd or did not finish untaring, so start again
		// should be idempotent, but rewriting buffers from JS is expensive, so just delete everything
		err := underlyingFs.Clear()
		if err != nil {
			return err
		}
	}

	fs, err := tarfs.New(r, underlyingFs)
	if err != nil {
		return err
	}
	go func() {
		<-fs.Done()
		err := fs.InitErr()
		if err != nil {
			log.Errorf("Failed to initialize mount %q: %v", mountPath, err)
			return
		}
		f, err := underlyingFs.Create(tarfsDoneMarker)
		if err != nil {
			log.Errorf("Failed to mark tarfs overlay %q complete: %v", mountPath, err)
			return
		}
		f.Close()
	}()
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
	return datasize.Bytes(total).String()
}
