package tarfs

import (
	"os"
	"time"
)

const dirSize = 42

type genericDirInfo struct {
	name string
}

func (g *genericDirInfo) Name() string {
	return g.name
}

func (g *genericDirInfo) Size() int64        { return dirSize }
func (g *genericDirInfo) Mode() os.FileMode  { return os.ModeDir | 0644 }
func (g *genericDirInfo) ModTime() time.Time { return time.Time{} }
func (g *genericDirInfo) IsDir() bool        { return true }
func (g *genericDirInfo) Sys() interface{}   { return nil }
