package fs

import (
	"fmt"
	"syscall/js"
)

type FID uint64

func (f FID) JSValue() js.Value {
	return js.ValueOf(uint64(f))
}

func (f *FID) String() string {
	if f == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *f)
}
