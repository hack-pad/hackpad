package fs

import (
	"context"

	"github.com/hack-pad/hackpadfs"
)

type clearFS interface {
	hackpadfs.FS
	Clear(ctx context.Context) error
}

type clearUnderlyingFS struct {
	hackpadfs.FS
	underlyingFS clearFS
}

func newClearUnderlyingFS(fs hackpadfs.FS, underlyingFS clearFS) clearFS {
	return &clearUnderlyingFS{
		FS:           fs,
		underlyingFS: underlyingFS,
	}
}

func (c *clearUnderlyingFS) Clear(ctx context.Context) error {
	return c.underlyingFS.Clear(ctx)
}
