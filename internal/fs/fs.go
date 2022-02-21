package fs

import (
	"compress/gzip"
	"context"
	"io"
	"path"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/cache"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/hack-pad/hackpadfs/mount"
	"github.com/hack-pad/hackpadfs/tar"
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
	if clearFs, ok := mount.(clearFS); ok {
		return clearFs.Clear(context.Background())
	}
	return &hackpadfs.PathError{Op: "clear", Path: path, Err: hackpadfs.ErrNotImplemented}
}

func Overlay(mountPath string, fs hackpadfs.FS) error {
	mountPath = common.ResolvePath(".", mountPath)
	return filesystem.AddMount(mountPath, fs)
}

type ShouldCacher func(name string, info hackpadfs.FileInfo) bool

func OverlayTarGzip(mountPath string, gzipReader io.ReadCloser, persist bool, shouldCache ShouldCacher) error {
	r, err := gzip.NewReader(gzipReader)
	if err != nil {
		return err
	}

	mountPath = common.ResolvePath(".", mountPath)
	if !persist {
		underlyingFS, err := mem.NewFS()
		if err != nil {
			return err
		}
		fs, err := tar.NewReaderFS(context.Background(), r, tar.ReaderFSOptions{
			UnarchiveFS: underlyingFS,
		})
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

	cacheOptions := cache.ReadOnlyOptions{
		RetainData: func(name string, info hackpadfs.FileInfo) bool {
			return shouldCache(path.Join(mountPath, name), info)
		},
	}
	newCacheFS := func(underlyingFS clearFS) (clearFS, error) {
		memFS, err := mem.NewFS()
		if err != nil {
			return nil, err
		}
		fs, err := cache.NewReadOnlyFS(underlyingFS, memFS, cacheOptions)
		if err != nil {
			return nil, err
		}
		return newClearUnderlyingFS(fs, underlyingFS), nil
	}

	_, err = hackpadfs.Stat(underlyingFS, tarfsDoneMarker)
	if err == nil {
		// tarfs already completed successfully and is persisted,
		// so close top-level reader and mount the existing files
		gzipReader.Close()

		cacheFS, err := newCacheFS(underlyingFS)
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

	readCtx, readCancel := context.WithCancel(context.Background())
	tarFS, err := tar.NewReaderFS(readCtx, r, tar.ReaderFSOptions{
		UnarchiveFS: underlyingFS,
	})
	if err != nil {
		readCancel()
		return err
	}
	tarClearFS := newClearCtxFS(underlyingFS, readCancel, tarFS.Done())
	cacheFS, err := newCacheFS(tarClearFS)
	if err != nil {
		return err
	}
	go func() {
		<-tarFS.Done()
		err := tarFS.UnarchiveErr()
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

type clearCtxFS struct {
	cancel context.CancelFunc
	wait   <-chan struct{}
	fs     clearFS
}

func newClearCtxFS(fs clearFS, cancel context.CancelFunc, wait <-chan struct{}) *clearCtxFS {
	return &clearCtxFS{
		cancel: cancel,
		wait:   wait,
		fs:     fs,
	}
}

func (c *clearCtxFS) Open(name string) (hackpadfs.File, error) {
	return c.fs.Open(name)
}

func (c *clearCtxFS) Clear(ctx context.Context) error {
	c.cancel()
	select {
	case <-c.wait:
		return c.fs.Clear(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
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
