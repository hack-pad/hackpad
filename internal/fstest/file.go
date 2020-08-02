package fstest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileClose(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileRead(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileReadAt(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileSeek(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileWrite(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileWriteAt(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileReaddir(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileReaddirnames(t *testing.T, undertest, expected FSTester) {
	require.NoError(t, expected.FS().Mkdir("foo", 0755))
	require.NoError(t, expected.FS().Mkdir("foo/bar", 0755))
	f, err := expected.FS().Create("foo/fizz")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = expected.FS().Create("foo/bar/baz")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	f, err = expected.FS().Open("foo")
	require.NoError(t, err)
	eNames1, err := f.Readdirnames(0)
	assert.NoError(t, err)
	eNames2, err := f.Readdirnames(-1)
	assert.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = expected.FS().Open("foo") // re-open to reset readdir count
	require.NoError(t, err)
	eNames3, err := f.Readdirnames(1)
	assert.NoError(t, err)
	eNames4, err := f.Readdirnames(1)
	assert.NoError(t, err)
	require.NoError(t, f.Close())
	expected.Clean()

	require.NoError(t, undertest.FS().Mkdir("foo", 0755))
	require.NoError(t, undertest.FS().Mkdir("foo/bar", 0755))
	f, err = undertest.FS().Create("foo/fizz")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = undertest.FS().Create("foo/bar/baz")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	f, err = undertest.FS().Open("foo")
	require.NoError(t, err)
	uNames1, err := f.Readdirnames(0)
	assert.NoError(t, err)
	uNames2, err := f.Readdirnames(-1)
	assert.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = undertest.FS().Open("foo") // re-open to reset readdir count
	require.NoError(t, err)
	uNames3, err := f.Readdirnames(1)
	assert.NoError(t, err)
	uNames4, err := f.Readdirnames(1)
	assert.NoError(t, err)
	require.NoError(t, f.Close())
	undertest.Clean()

	assert.Equal(t, eNames1, uNames1)
	assert.Equal(t, eNames2, uNames2)
	assert.Equal(t, eNames3, uNames3)
	assert.Equal(t, eNames4, uNames4)
}

func TestFileStat(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileSync(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileTruncate(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFileWriteString(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}
