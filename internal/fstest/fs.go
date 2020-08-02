package fstest

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsBasicCreate(t *testing.T, undertest, expected FSTester) {
	eFile, eErr := expected.FS().Create("foo")
	expected.Clean()
	uFile, uErr := undertest.FS().Create("foo")
	assert.Equal(t, eErr, uErr)
	assert.NotNil(t, eFile)
	assert.NotNil(t, uFile)
}

// Mkdir creates a new directory with the specified name and permission bits (before umask). If there is an error, it will be of type *PathError.
func TestFsBasicMkdir(t *testing.T, undertest, expected FSTester) {
	eErr := expected.FS().Mkdir("foo", 0600)
	expected.Clean()

	uErr := undertest.FS().Mkdir("foo", 0600)
	assert.Equal(t, eErr, uErr)
	undertest.Clean()
}

func TestFsCreate(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Mkdir creates a new directory with the specified name and permission bits (before umask). If there is an error, it will be of type *PathError.
func TestFsMkdir(t *testing.T, undertest, expected FSTester) {
	t.Run("fail dir exists", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0600))
		eErr := expected.FS().Mkdir("foo", 0600)
		assert.Error(t, eErr)
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0600))
		uErr := undertest.FS().Mkdir("foo", 0600)
		assert.Error(t, uErr)
		assertEqualFS(t, expectedStat, statFS(t, undertest.FS()))
		undertest.Clean()

		assert.True(t, os.IsExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "mkdir", pathErr.Op)
	})

	t.Run("fail file exists", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		eErr := expected.FS().Mkdir("foo", 0600)
		assert.Error(t, eErr)
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		uErr := undertest.FS().Mkdir("foo", 0600)
		assert.Error(t, uErr)
		assertEqualFS(t, expectedStat, statFS(t, undertest.FS()))
		undertest.Clean()

		assert.True(t, os.IsExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "mkdir", pathErr.Op)
	})

	t.Run("create sub dir", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		require.NoError(t, expected.FS().Mkdir("foo/bar", 0600))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().Mkdir("foo", 0700))
		assert.NoError(t, undertest.FS().Mkdir("foo/bar", 0600))

		assertEqualFS(t, expectedStat, statFS(t, undertest.FS()))
		undertest.Clean()
	})

	t.Run("only permission bits allowed", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", os.ModeSocket|0755))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().Mkdir("foo", os.ModeSocket|0755))
		assertEqualFS(t, expectedStat, statFS(t, undertest.FS()))
		undertest.Clean()
	})
}

func TestFsMkdirAll(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsOpen(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsOpenFile(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsRemove(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsRemoveAll(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsRename(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsStat(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsChmod(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsChtimes(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}
