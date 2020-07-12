package fs

import (
	"archive/zip"

	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
)

func OverlayZip(z *zip.Reader) {
	filesystem = afero.NewCopyOnWriteFs(zipfs.New(z), filesystem)
}
