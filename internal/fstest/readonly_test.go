package fstest

import (
	"testing"

	"github.com/spf13/afero"
)

func TestReadOnly(t *testing.T) {
	fs := afero.NewMemMapFs()
	RunReadOnly(t, afero.NewReadOnlyFs(fs), fs, cleanUpMemFs(fs))
}
