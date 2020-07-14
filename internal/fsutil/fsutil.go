package fsutil

import (
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

func NormalizePath(path string) string {
	if !strings.HasPrefix(path, afero.FilePathSeparator) {
		path = filepath.Join(afero.FilePathSeparator, path) // prepend "/" to ensure "/tmp" and "tmp" are identical files
	} else {
		path = filepath.Clean(path)
	}

	switch path {
	case ".":
		return afero.FilePathSeparator
	case "..":
		return afero.FilePathSeparator
	default:
		return path
	}
}
