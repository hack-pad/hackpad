//go:build !js
// +build !js

package fs

import (
	"context"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
)

type persistFsInterface interface {
	hackpadfs.FS
	hackpadfs.ChmodFS
	hackpadfs.MkdirFS
	hackpadfs.OpenFileFS
}

type persistFs struct {
	persistFsInterface
}

func newPersistDB(name string, relaxedDurability bool, shouldCache ShouldCacher) (*persistFs, error) {
	panic("not implemented")
}

func (p *persistFs) Clear(context.Context) error {
	panic("not implemented")
}

func newBlobLength(i int) (blob.Blob, error) {
	return blob.NewBytesLength(i), nil
}
