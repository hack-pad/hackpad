package common

import (
	"path"
	"strings"
)

func ResolvePath(wd, p string) string {
	if path.IsAbs(p) {
		p = path.Clean(p)
	} else {
		p = path.Join(wd, p)
	}
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "."
	}
	return p
}
