package fs

import (
	"context"
	"io"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/cache"
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

type ShouldCacher func(name string, info hackpadfs.FileInfo) bool

func OverlayTarGzip(mountPath string, r io.ReadCloser, persist bool, shouldCache ShouldCacher) error {
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

	underlyingFS, err := newPersistDB(mountPath, true, shouldCache)
	if err != nil {
		return err
	}

	memFS, err := mem.NewFS()
	if err != nil {
		return err
	}

	cacheOptions := cache.ReadOnlyOptions{
		RetainData: func(name string, info hackpadfs.FileInfo) bool {
			return shouldCache(path.Join(mountPath, name), info)
		},
	}

	_, err = hackpadfs.Stat(underlyingFS, tarfsDoneMarker)
	if err == nil {
		// tarfs already completed successfully and is persisted,
		// so close tarfs reader and mount the existing files
		r.Close()

		cacheFS, err := cache.NewReadOnlyFS(underlyingFS, memFS, cacheOptions)
		if err != nil {
			return err
		}
		return filesystem.AddMount(mountPath, cacheFS)
	} else {
		// either never untar'd or did not finish untaring, so start again
		// should be idempotent, but rewriting buffers from JS is expensive, so just delete everything
		err := underlyingFS.Clear(context.Background())
		if err != nil {
			return err
		}
	}

	tarFS, err := tarfs.New(r, underlyingFS)
	if err != nil {
		return err
	}
	cacheFS, err := cache.NewReadOnlyFS(tarFS, memFS, cacheOptions)
	if err != nil {
		return err
	}
	go func() {
		<-tarFS.Done()
		err := tarFS.InitErr()
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
	return filesystem.AddMount(mountPath, cacheFS)
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
