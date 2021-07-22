package fs

import (
	"context"
	"io"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/tarfs"
	"github.com/johnstarich/go-wasm/log"
	"github.com/johnstarich/go/datasize"
)

var (
	filesystem = func() rootFs {
		memFS, err := mem.NewFS()
		if err != nil {
			panic(err)
		}
		fs, err := mount.NewFS(memFS)
		if err != nil {
			panic(err)
		}
		return fs
	}()
)

type rootFs interface {
	hackpadfs.MountFS
	AddMount(path string, mount hackpadfs.FS) error
	MountPoints() []mount.Point
}

func Mounts() []mount.Point {
	return filesystem.MountPoints()
}

func DestroyMount(path string) error {
	mount, _ := filesystem.Mount(path)
	if clearFs, ok := mount.(interface{ Clear() error }); ok {
		return clearFs.Clear()
	}
	return &hackpadfs.PathError{Op: "clear", Path: path, Err: hackpadfs.ErrNotImplemented}
}

func Overlay(mountPath string, fs hackpadfs.FS) error {
	mountPath = common.ResolvePath(".", mountPath)
	return filesystem.AddMount(mountPath, fs)
}

type ShouldCacher func(string) bool

func OverlayTarGzip(mountPath string, r io.ReadCloser, persist bool) error {
	mountPath = common.ResolvePath(".", mountPath)
	if !persist {
		underlyingFS, err := mem.NewFS()
		if err != nil {
			return err
		}
		fs, err := tarfs.New(r, underlyingFS)
		if err != nil {
			return err
		}
		return filesystem.AddMount(mountPath, fs)
	}

	const tarfsDoneMarker = ".tarfs-complete"

	underlyingFS, err := newPersistDB(mountPath, true, func(string) bool { return true })
	if err != nil {
		return err
	}

	_, err = underlyingFS.Stat(tarfsDoneMarker)
	if err == nil {
		// tarfs already completed successfully and is persisted,
		// so close tarfs reader and mount the existing files
		r.Close()

		type readOnly struct {
			hackpadfs.FS
		}

		return filesystem.AddMount(mountPath, &readOnly{underlyingFS})
	} else {
		// either never untar'd or did not finish untaring, so start again
		// should be idempotent, but rewriting buffers from JS is expensive, so just delete everything
		err := underlyingFS.Clear(context.Background())
		if err != nil {
			return err
		}
	}

	fs, err := tarfs.New(r, underlyingFS)
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
		f, err := hackpadfs.Create(underlyingFS, tarfsDoneMarker)
		if err != nil {
			log.Errorf("Failed to mark tarfs overlay %q complete: %v", mountPath, err)
			return
		}
		f.Close()
	}()
	return filesystem.AddMount(mountPath, fs)
}

// Dump prints out file system statistics
func Dump(basePath string) interface{} {
	var total int64
	err := hackpadfs.WalkDir(filesystem, basePath, func(path string, dirEntry hackpadfs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := dirEntry.Info()
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
