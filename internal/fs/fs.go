package fs

import (
	"archive/zip"
	"fmt"
	"os"

	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
)

var (
	filesystem = afero.NewMemMapFs()
)

func OverlayZip(z *zip.Reader) {
	filesystem = afero.NewCopyOnWriteFs(zipfs.New(z), filesystem)
}

func Dump(basePath string) interface{} {
	var total int64
	err := afero.Walk(filesystem, basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	if err != nil {
		return err
	}
	return fmt.Sprintf("%d bytes", total)
}
