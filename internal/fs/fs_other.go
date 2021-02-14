// +build !js

package fs

import "github.com/spf13/afero"

type persistFs struct {
	afero.Fs
}

func newPersistDB(name string) (*persistFs, error) {
	panic("not implemented")
}

func (p *persistFs) Clear() error {
	panic("not implemented")
}
