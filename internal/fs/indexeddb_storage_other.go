// +build !js

package fs

import "github.com/spf13/afero"

func NewIndexedDBFs(name string) (_ afero.Fs, err error) {
	panic("Not supported outside of JS")
}
