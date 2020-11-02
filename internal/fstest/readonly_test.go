// +build !js

package fstest

import (
	"testing"

	"github.com/spf13/afero"
)

func TestReadOnly(t *testing.T) {
	fs := afero.NewMemMapFs()
	commitWrites := func() error { return nil }
	RunReadOnly(t, afero.NewReadOnlyFs(fs), fs, cleanUpMemFs(fs), commitWrites)
}
