package common

import (
	"fmt"
	"io"

	"github.com/hack-pad/hackpadfs"
)

type FID uint64

func (f *FID) String() string {
	if f == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *f)
}

type OpenFileAttr struct {
	FilePath   string
	SeekOffset int64
	Flags      int
	Mode       hackpadfs.FileMode
	RawDevice io.ReadWriteCloser
}
