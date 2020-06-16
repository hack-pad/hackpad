package fs

import "syscall/js"

type FID uint64

func (f FID) JSValue() js.Value {
	return js.ValueOf(uint64(f))
}
