package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type stater func(string) (os.FileInfo, error)

func lookPath(stat stater, pathVar string, file string) (string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(stat, file)
		if err == nil {
			return file, nil
		}
		return "", &exec.Error{Name: file, Err: err}
	}
	for _, dir := range filepath.SplitList(pathVar) {
		if dir == "" {
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(stat, path); err == nil {
			return path, nil
		}
	}
	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

func findExecutable(stat stater, file string) error {
	d, err := stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}
