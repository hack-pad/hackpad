package fs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/spf13/afero"
)

var (
	filesystem = afero.NewMemMapFs()
)

// Dump prints out file system statistics
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

// DumpZip starts a zip download of everything in the given directory
func DumpZip(path string) error {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	err := afero.Walk(filesystem, path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		zipInfo, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		zipInfo.Name = path
		if info.IsDir() {
			zipInfo.Name += "/"
		}
		w, err := z.CreateHeader(zipInfo)
		if err != nil {
			return err
		}
		r, err := filesystem.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(w, r)
		r.Close()
		return err
	})
	if err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	interop.StartDownload("application/zip", strings.ReplaceAll(path, string(filepath.Separator), "-")+".zip", buf.Bytes())
	return nil
}
