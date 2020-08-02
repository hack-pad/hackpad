package fstest

import (
	"testing"

	"github.com/spf13/afero"
)

func TestBaseline(t *testing.T) {
	Run(t, afero.NewOsFs(), cleanUpOsFs)
}

func TestMemMapFs(t *testing.T) {
	fs := afero.NewMemMapFs()
	Run(t, fs, func() error {
		root, err := fs.Open("/")
		if err != nil {
			return err
		}
		defer root.Close()
		names, err := root.Readdirnames(-1)
		if err != nil {
			return err
		}
		for _, name := range names {
			if err := fs.RemoveAll(name); err != nil {
				return err
			}
		}
		return nil
	})
}
