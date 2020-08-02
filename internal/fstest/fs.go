package fstest

import (
	"io"
	"os"
	"strings"
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

func TestFsBasicMkdir(t *testing.T, undertest, expected FSTester) {
	eErr := expected.FS().Mkdir("foo", 0600)
	expected.Clean()

	uErr := undertest.FS().Mkdir("foo", 0600)
	assert.Equal(t, eErr, uErr)
	undertest.Clean()
}

// Create creates or truncates the named file.
// If the file already exists, it is truncated.
// If the file does not exist, it is created with mode 0666 (before umask).
// If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
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
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
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
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
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

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing and returns nil.
func TestFsMkdirAll(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Open opens the named file for reading.
// If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func TestFsOpen(t *testing.T, undertest, expected FSTester) {
	t.Run("does not exist", func(t *testing.T) {
		_, eErr := expected.FS().Open("foo")
		assert.Error(t, eErr)
		_, uErr := undertest.FS().Open("foo")
		assert.Error(t, uErr)

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("open file", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = expected.FS().Open("foo")
		assert.NoError(t, err)
		assert.NotNil(t, f)
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = undertest.FS().Open("foo")
		assert.NoError(t, err)
		assert.NotNil(t, f)
		require.NoError(t, f.Close())
		undertest.Clean()
	})

	t.Run("supports reads", func(t *testing.T) {
		const fileContents = `hello world`
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		n, err := f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = expected.FS().Open("foo")
		assert.NoError(t, err)
		buf := make([]byte, n)
		n2, err := io.ReadFull(f, buf)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
		assert.Equal(t, fileContents, string(buf))
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		n, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = undertest.FS().Open("foo")
		assert.NoError(t, err)
		buf = make([]byte, n)
		n2, err = io.ReadFull(f, buf)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
		assert.Equal(t, fileContents, string(buf))
		undertest.Clean()
	})

	t.Run("fails writes", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = expected.FS().Open("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(`bar`))
		assert.Error(t, err)
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = undertest.FS().Open("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(`bar`))
		assert.Error(t, err)
		undertest.Clean()
	})
}

// OpenFile is the generalized open call; most users will use Open or Create instead.
// It opens the named file with specified flag (O_RDONLY etc.).
// If the file does not exist, and the O_CREATE flag is passed, it is created with mode perm (before umask).
// If successful, methods on the returned File can be used for I/O.
// If there is an error, it will be of type *PathError.
func TestFsOpenFile(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Remove removes the named file or (empty) directory. If there is an error, it will be of type *PathError.
func TestFsRemove(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error it encounters.
// If the path does not exist, RemoveAll returns nil (no error).
// If there is an error, it will be of type *PathError.
func TestFsRemoveAll(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func TestFsRename(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Stat returns a FileInfo describing the named file. If there is an error, it will be of type *PathError.
func TestFsStat(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
//
// A different subset of the mode bits are used, depending on the operating system.
//
// fstest will only check permission bits
func TestFsChmod(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}

func TestFsChtimes(t *testing.T, undertest, expected FSTester) {
	t.Skip()
}
