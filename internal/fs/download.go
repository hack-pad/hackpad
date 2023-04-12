//go:build js
// +build js

package fs

import (
	"archive/zip"
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpadfs"
)

// DumpZip starts a zip download of everything in the given directory
func DumpZip(path string) error {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	err := hackpadfs.WalkDir(filesystem, path, func(path string, dirEntry hackpadfs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := dirEntry.Info()
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
