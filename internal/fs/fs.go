package fs

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
)

func Dump() interface{} {
	var total int64
	err := afero.Walk(filesystem, "/", func(path string, info os.FileInfo, err error) error {
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
