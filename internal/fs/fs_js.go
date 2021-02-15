// +build js

package fs

import (
	"github.com/johnstarich/go-wasm/internal/nativeio"
	"github.com/spf13/afero"
)

type persistFs interface {
	afero.Fs
	Clear() error
}

func newPersistFs(name string) (persistFs, error) {
	if nativeio.Supported() {
		return NewNativeIOFs(name)
	}
	return NewIndexedDBFs(name)
}
