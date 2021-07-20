package fsutil

import "path"

func NormalizePath(p string) string {
	return path.Clean(p)
}
