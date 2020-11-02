package common

import (
	"fmt"
)

type FID uint64

func (f *FID) String() string {
	if f == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%d", *f)
}
