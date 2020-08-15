package fstest

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsBasicCreate(t *testing.T, undertest, expected FSTester) {
	file, err := expected.FS().Create("foo")
	require.NoError(t, err)
	assert.NotNil(t, file)
	require.NoError(t, file.Close())
	expected.Clean()

	file, err = undertest.FS().Create("foo")
	require.NoError(t, err)
	assert.NotNil(t, file)
	require.NoError(t, file.Close())
}

func TestFsBasicMkdir(t *testing.T, undertest, expected FSTester) {
	eErr := expected.FS().Mkdir("foo", 0600)
	expected.Clean()

	uErr := undertest.FS().Mkdir("foo", 0600)
	assert.Equal(t, eErr, uErr)
	undertest.Clean()
}

func TestFsBasicChmod(t *testing.T, undertest, expected FSTester) {
	f, err := expected.FS().Create("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = expected.FS().Chmod("foo", 0755)
	assert.NoError(t, err)
	expected.Clean()

	f, err = undertest.FS().Create("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = undertest.FS().Chmod("foo", 0755)
	assert.NoError(t, err)
	undertest.Clean()
}

func TestFsBasicChtimes(t *testing.T, undertest, expected FSTester) {
	var (
		accessTime = time.Now()
		modifyTime = accessTime.Add(-10 * time.Second)
	)

	f, err := expected.FS().Create("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = expected.FS().Chtimes("foo", accessTime, modifyTime)
	assert.NoError(t, err)
	expected.Clean()

	f, err = undertest.FS().Create("foo")
	require.NoError(t, err)
	require.NoError(t, f.Close())

	err = undertest.FS().Chtimes("foo", accessTime, modifyTime)
	assert.NoError(t, err)
	undertest.Clean()
}

// Create creates or truncates the named file.
// If the file already exists, it is truncated.
// If the file does not exist, it is created with mode 0666 (before umask).
// If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
// If there is an error, it will be of type *PathError.
func TestFsCreate(t *testing.T, undertest, expected FSTester) {
	testFsCreate(t, undertest, expected, func(fs afero.Fs, name string) (afero.File, error) {
		return fs.Create(name)
	})
}

func testFsCreate(t *testing.T, undertest, expected FSTester, createFn func(afero.Fs, string) (afero.File, error)) {
	t.Run("new file", func(t *testing.T) {
		file, err := createFn(expected.FS(), "foo")
		require.NoError(t, err)
		assert.NotNil(t, file)
		require.NoError(t, file.Close())
		eInfo, err := expected.FS().Stat("foo")
		require.NoError(t, err)
		expected.Clean()

		file, err = createFn(undertest.FS(), "foo")
		require.NoError(t, err)
		assert.NotNil(t, file)
		require.NoError(t, file.Close())
		uInfo, err := undertest.FS().Stat("foo")
		require.NoError(t, err)
		undertest.Clean()

		assert.Equal(t, os.FileMode(0666).String(), uInfo.Mode().String())
		assertEqualFileInfo(t, eInfo, uInfo)
	})

	t.Run("existing file", func(t *testing.T) {
		const fileContents = `hello world`

		file, err := createFn(expected.FS(), "foo")
		require.NoError(t, err)
		_, err = file.Write([]byte(fileContents))
		assert.NoError(t, err)
		require.NoError(t, file.Close())
		require.NoError(t, expected.FS().Chmod("foo", 0755))

		file, err = createFn(expected.FS(), "foo")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		eInfo, err := expected.FS().Stat("foo")
		require.NoError(t, err)
		expected.Clean()

		file, err = createFn(undertest.FS(), "foo")
		require.NoError(t, err)
		_, err = file.Write([]byte(fileContents))
		assert.NoError(t, err)
		require.NoError(t, file.Close())
		require.NoError(t, undertest.FS().Chmod("foo", 0755))

		file, err = createFn(undertest.FS(), "foo")
		require.NoError(t, err)
		require.NoError(t, file.Close())
		uInfo, err := undertest.FS().Stat("foo")
		require.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eInfo, uInfo)
	})

	t.Run("existing directory", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		_, eErr := createFn(expected.FS(), "foo")
		assert.Error(t, eErr)
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		_, uErr := createFn(undertest.FS(), "foo")
		assert.Error(t, uErr)
		undertest.Clean()

		assert.True(t, afero.IsDirErr(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("parent directory must exist", func(t *testing.T) {
		_, err := createFn(expected.FS(), "foo/bar")
		assert.Error(t, err)
		expected.Clean()

		_, uErr := createFn(undertest.FS(), "foo/bar")
		assert.Error(t, uErr)
		undertest.Clean()

		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo/bar", strings.TrimPrefix(pathErr.Path, "/"))
	})
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

	t.Run("parent directory must exist", func(t *testing.T) {
		assert.Error(t, expected.FS().Mkdir("foo/bar", 0755))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		uErr := undertest.FS().Mkdir("foo/bar", 0755)
		assert.Error(t, uErr)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "mkdir", pathErr.Op)
		assert.Equal(t, "foo/bar", strings.TrimPrefix(pathErr.Path, "/"))

		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing and returns nil.
func TestFsMkdirAll(t *testing.T, undertest, expected FSTester) {
	t.Run("create one directory", func(t *testing.T) {
		assert.NoError(t, expected.FS().MkdirAll("foo", 0700))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().MkdirAll("foo", 0700))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("create multiple directories", func(t *testing.T) {
		assert.NoError(t, expected.FS().MkdirAll("foo/bar", 0700))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().MkdirAll("foo/bar", 0700))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("all directories exist", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		require.NoError(t, expected.FS().Mkdir("foo/bar", 0644))
		assert.NoError(t, expected.FS().MkdirAll("foo/bar", 0600))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		require.NoError(t, undertest.FS().Mkdir("foo/bar", 0644))
		assert.NoError(t, undertest.FS().MkdirAll("foo/bar", 0600))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("file exists", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.Error(t, expected.FS().MkdirAll("foo/bar", 0700))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		uErr := undertest.FS().MkdirAll("foo/bar", 0700)
		assert.Error(t, uErr)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.True(t, afero.IsNotDir(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "mkdir", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("illegal permission bits", func(t *testing.T) {
		assert.NoError(t, expected.FS().MkdirAll("foo/bar", os.ModeSocket|0777))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().MkdirAll("foo/bar", os.ModeSocket|0777))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// Open opens the named file for reading.
// If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
// If there is an error, it will be of type *PathError.
func TestFsOpen(t *testing.T, undertest, expected FSTester) {
	testFsOpen(t, undertest, expected, func(fs afero.Fs, name string) (afero.File, error) {
		return fs.Open(name)
	})
}

func testFsOpen(t *testing.T, undertest, expected FSTester, openFn func(afero.Fs, string) (afero.File, error)) {
	t.Run("does not exist", func(t *testing.T) {
		_, eErr := openFn(expected.FS(), "foo")
		assert.Error(t, eErr)
		_, uErr := openFn(undertest.FS(), "foo")
		assert.Error(t, uErr)

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("open file", func(t *testing.T) {
		f, err := expected.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(expected.FS(), "foo")
		assert.NoError(t, err)
		assert.NotNil(t, f)
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(undertest.FS(), "foo")
		assert.NoError(t, err)
		assert.NotNil(t, f)
		require.NoError(t, f.Close())
		undertest.Clean()
	})

	t.Run("supports reads", func(t *testing.T) {
		const fileContents = `hello world`
		f, err := expected.WriteFS().Create("foo")
		require.NoError(t, err)
		n, err := f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(expected.FS(), "foo")
		assert.NoError(t, err)
		buf := make([]byte, n)
		n2, err := io.ReadFull(f, buf)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
		assert.Equal(t, fileContents, string(buf))
		expected.Clean()

		f, err = undertest.WriteFS().Create("foo")
		require.NoError(t, err)
		n, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(undertest.FS(), "foo")
		assert.NoError(t, err)
		buf = make([]byte, n)
		n2, err = io.ReadFull(f, buf)
		assert.NoError(t, err)
		assert.Equal(t, n, n2)
		assert.Equal(t, fileContents, string(buf))
		undertest.Clean()
	})

	t.Run("fails writes", func(t *testing.T) {
		f, err := expected.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(expected.FS(), "foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(`bar`))
		assert.Error(t, err)
		expected.Clean()

		f, err = undertest.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		f, err = openFn(undertest.FS(), "foo")
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
	t.Run("read-only", func(t *testing.T) {
		testFsOpen(t, undertest, expected, func(fs afero.Fs, name string) (afero.File, error) {
			return fs.OpenFile(name, os.O_RDONLY, 0777)
		})
	})

	t.Run("create", func(t *testing.T) {
		testFsCreate(t, undertest, expected, func(fs afero.Fs, name string) (afero.File, error) {
			return fs.OpenFile(name, os.O_RDWR|os.O_CREATE, 0666)
		})
	})

	t.Run("create illegal perms", func(t *testing.T) {
		f, err := expected.FS().OpenFile("foo", os.O_RDWR|os.O_CREATE, os.ModeSocket|0777)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().OpenFile("foo", os.O_RDWR|os.O_CREATE, os.ModeSocket|0777)
		require.NoError(t, err)
		require.NoError(t, f.Close())
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("truncate on existing file", func(t *testing.T) {
		const fileContents = "hello world"

		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		_, err = expected.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.NoError(t, err)
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		_, err = undertest.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.NoError(t, err)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("truncate on non-existent file", func(t *testing.T) {
		_, err := expected.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.Error(t, err)
		expected.Clean()

		_, uErr := undertest.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.Error(t, uErr)
		undertest.Clean()

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("truncate on existing dir", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		_, err := expected.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.Error(t, err)
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		_, uErr := undertest.FS().OpenFile("foo", os.O_TRUNC, 0700)
		assert.Error(t, err)
		undertest.Clean()

		assert.True(t, afero.IsDirErr(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "open", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("append flag writes to end", func(t *testing.T) {
		const (
			fileContents1 = "hello world"
			fileContents2 = "sup "
		)

		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents1))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		f, err = expected.FS().OpenFile("foo", os.O_RDWR|os.O_APPEND, 0700)
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents2))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents1))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		f, err = undertest.FS().OpenFile("foo", os.O_RDWR|os.O_APPEND, 0700)
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents2))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// Remove removes the named file or (empty) directory. If there is an error, it will be of type *PathError.
func TestFsRemove(t *testing.T, undertest, expected FSTester) {
	t.Run("remove file", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Remove("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Remove("foo"))
		undertestStat := statFS(t, expected.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove empty dir", func(t *testing.T) {
		err := expected.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		assert.NoError(t, expected.FS().Remove("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		err = undertest.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		assert.NoError(t, undertest.FS().Remove("foo"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove non-existing file", func(t *testing.T) {
		assert.Error(t, expected.FS().Remove("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		uErr := undertest.FS().Remove("foo")
		assert.Error(t, uErr)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "remove", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove non-empty dir", func(t *testing.T) {
		err := expected.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		f, err := expected.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.Error(t, expected.FS().Remove("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		err = undertest.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		f, err = undertest.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		uErr := undertest.FS().Remove("foo")
		assert.Error(t, uErr)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.True(t, os.IsExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "remove", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error it encounters.
// If the path does not exist, RemoveAll returns nil (no error).
// If there is an error, it will be of type *PathError.
func TestFsRemoveAll(t *testing.T, undertest, expected FSTester) {
	t.Run("remove file", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().RemoveAll("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().RemoveAll("foo"))
		undertestStat := statFS(t, expected.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove empty dir", func(t *testing.T) {
		err := expected.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		assert.NoError(t, expected.FS().RemoveAll("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		err = undertest.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		assert.NoError(t, undertest.FS().RemoveAll("foo"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove non-existing file", func(t *testing.T) {
		assert.NoError(t, expected.FS().RemoveAll("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		assert.NoError(t, undertest.FS().RemoveAll("foo"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("remove non-empty dir", func(t *testing.T) {
		err := expected.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		f, err := expected.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().RemoveAll("foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		err = undertest.FS().Mkdir("foo", 0700)
		require.NoError(t, err)
		f, err = undertest.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().RemoveAll("foo"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// Rename renames (moves) oldpath to newpath.
// If newpath already exists and is not a directory, Rename replaces it.
// OS-specific restrictions may apply when oldpath and newpath are in different directories.
// If there is an error, it will be of type *LinkError.
func TestFsRename(t *testing.T, undertest, expected FSTester) {
	t.Run("inside same directory", func(t *testing.T) {
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		f, err := expected.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Rename("foo/bar", "foo/baz"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		f, err = undertest.FS().Create("foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Rename("foo/bar", "foo/baz"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("inside same directory in root", func(t *testing.T) {
		f, err := expected.FS().Create("bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Rename("bar", "baz"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		f, err = undertest.FS().Create("bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Rename("bar", "baz"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("same file", func(t *testing.T) {
		const fileContents = `hello world`
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		f, err := expected.FS().Create("foo/bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Rename("foo/bar", "foo/bar"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		f, err = undertest.FS().Create("foo/bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Rename("foo/bar", "foo/bar"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("same directory", func(t *testing.T) {
		const fileContents = `hello world`
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		assert.Error(t, expected.FS().Rename("foo", "foo"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		uErr := undertest.FS().Rename("foo", "foo")
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.Error(t, uErr)
		assert.True(t, os.IsExist(uErr))
		require.IsType(t, &os.LinkError{}, uErr)
		linkErr := uErr.(*os.LinkError)
		assert.Equal(t, "rename", linkErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(linkErr.Old, "/"))
		assert.Equal(t, "foo", strings.TrimPrefix(linkErr.New, "/"))

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("newpath is directory", func(t *testing.T) {
		const fileContents = `hello world`
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		require.NoError(t, expected.FS().Mkdir("bar", 0700))
		assert.Error(t, expected.FS().Rename("foo", "bar"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		require.NoError(t, undertest.FS().Mkdir("bar", 0700))
		uErr := undertest.FS().Rename("foo", "bar")
		assert.Error(t, uErr)
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assert.Error(t, uErr)
		assert.True(t, os.IsExist(uErr))
		require.IsType(t, &os.LinkError{}, uErr)
		linkErr := uErr.(*os.LinkError)
		assert.Equal(t, "rename", linkErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(linkErr.Old, "/"))
		assert.Equal(t, "bar", strings.TrimPrefix(linkErr.New, "/"))

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("newpath in root", func(t *testing.T) {
		const fileContents = `hello world`
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		f, err := expected.FS().Create("foo/bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Rename("foo/bar", "baz"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		f, err = undertest.FS().Create("foo/bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Rename("foo/bar", "baz"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})

	t.Run("newpath in subdirectory", func(t *testing.T) {
		const fileContents = `hello world`
		require.NoError(t, expected.FS().Mkdir("foo", 0700))
		f, err := expected.FS().Create("bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, expected.FS().Rename("bar", "foo/baz"))
		expectedStat := statFS(t, expected.FS())
		expected.Clean()

		require.NoError(t, undertest.FS().Mkdir("foo", 0700))
		f, err = undertest.FS().Create("bar")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		assert.NoError(t, undertest.FS().Rename("bar", "foo/baz"))
		undertestStat := statFS(t, undertest.FS())
		undertest.Clean()

		assertEqualFS(t, expectedStat, undertestStat)
	})
}

// Stat returns a FileInfo describing the named file. If there is an error, it will be of type *PathError.
func TestFsStat(t *testing.T, undertest, expected FSTester) {
	testStat(t, undertest, expected, func(tb testing.TB, fsTest FSTester, path string) (os.FileInfo, error) {
		return fsTest.FS().Stat(path)
	})
}

func testStat(t *testing.T, undertest, expected FSTester, stater func(testing.TB, FSTester, string) (os.FileInfo, error)) {
	t.Run("stat a file", func(t *testing.T) {
		f, err := expected.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, expected.FS().Chmod("foo", 0755))
		eInfo, err := stater(t, expected, "foo")
		assert.NoError(t, err)
		expected.Clean()

		f, err = undertest.WriteFS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, undertest.WriteFS().Chmod("foo", 0755))
		uInfo, err := stater(t, undertest, "foo")
		assert.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eInfo, uInfo)
	})

	t.Run("stat a directory", func(t *testing.T) {
		err := expected.WriteFS().Mkdir("foo", 0755)
		require.NoError(t, err)
		eInfo, err := stater(t, expected, "foo")
		assert.NoError(t, err)
		expected.Clean()

		err = undertest.WriteFS().Mkdir("foo", 0755)
		require.NoError(t, err)
		uInfo, err := stater(t, undertest, "foo")
		assert.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eInfo, uInfo)
	})

	t.Run("stat nested files", func(t *testing.T) {
		err := expected.WriteFS().Mkdir("foo", 0755)
		require.NoError(t, err)
		err = expected.WriteFS().Mkdir("foo/bar", 0755)
		require.NoError(t, err)
		f, err := expected.WriteFS().Create("foo/bar/baz")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		eInfo1, err := stater(t, expected, "foo/bar")
		assert.NoError(t, err)
		eInfo2, err := stater(t, expected, "foo/bar/baz")
		assert.NoError(t, err)
		expected.Clean()

		err = undertest.WriteFS().Mkdir("foo", 0755)
		require.NoError(t, err)
		err = undertest.WriteFS().Mkdir("foo/bar", 0755)
		require.NoError(t, err)
		f, err = undertest.WriteFS().Create("foo/bar/baz")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		uInfo1, err := stater(t, undertest, "foo/bar")
		assert.NoError(t, err)
		uInfo2, err := stater(t, undertest, "foo/bar/baz")
		assert.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eInfo1, uInfo1)
		assertEqualFileInfo(t, eInfo2, uInfo2)
	})
}

// Chmod changes the mode of the named file to mode.
// If the file is a symbolic link, it changes the mode of the link's target.
// If there is an error, it will be of type *PathError.
//
// A different subset of the mode bits are used, depending on the operating system.
//
// fstest will only check permission bits
func TestFsChmod(t *testing.T, undertest, expected FSTester) {
	t.Run("change permission bits", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = expected.FS().Chmod("foo", 0755)
		assert.NoError(t, err)
		eInfo, err := expected.FS().Stat("foo")
		require.NoError(t, err)
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = undertest.FS().Chmod("foo", 0755)
		assert.NoError(t, err)
		uInfo, err := undertest.FS().Stat("foo")
		require.NoError(t, err)
		undertest.Clean()
		assertEqualFileInfo(t, eInfo, uInfo)
	})

	uLinker, uOK := undertest.FS().(afero.Symlinker)
	eLinker, eOK := expected.FS().(afero.Symlinker)
	if !uOK {
		t.Skip("Skipping symlink tests, 'undertest' does not support afero.Symlinker")
	}
	if !eOK {
		t.Skip("Skipping symlink tests, 'expected' does not support afero.Symlinker")
	}

	t.Run("change symlink targets permission bits", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, eLinker.SymlinkIfPossible("foo", "bar"))

		err = expected.FS().Chmod("foo", 0755)
		assert.NoError(t, err)
		eLinkInfo, err := expected.FS().Stat("foo")
		require.NoError(t, err)
		eInfo, err := expected.FS().Stat("bar")
		require.NoError(t, err)
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, uLinker.SymlinkIfPossible("foo", "bar"))

		err = undertest.FS().Chmod("foo", 0755)
		assert.NoError(t, err)
		uLinkInfo, err := undertest.FS().Stat("foo")
		require.NoError(t, err)
		uInfo, err := undertest.FS().Stat("bar")
		require.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eLinkInfo, uLinkInfo)
		assertEqualFileInfo(t, eInfo, uInfo)
	})
}

// Chtimes changes the access and modification times of the named file, similar to the Unix utime() or utimes() functions.
//
// The underlying filesystem may truncate or round the values to a less precise time unit. If there is an error, it will be of type *PathError.
func TestFsChtimes(t *testing.T, undertest, expected FSTester) {
	var (
		accessTime = time.Now()
		modifyTime = accessTime.Add(-1 * time.Minute)
	)

	t.Run("file does not exist", func(t *testing.T) {
		eErr := expected.FS().Chtimes("foo", accessTime, modifyTime)
		assert.Error(t, eErr)
		expected.Clean()

		uErr := undertest.FS().Chtimes("foo", accessTime, modifyTime)
		assert.Error(t, uErr)
		undertest.Clean()

		assert.True(t, os.IsNotExist(uErr))
		require.IsType(t, &os.PathError{}, uErr)
		pathErr := uErr.(*os.PathError)
		assert.Equal(t, "chtimes", pathErr.Op)
		assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
	})

	t.Run("change access and modify times", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = expected.FS().Chtimes("foo", accessTime, modifyTime)
		assert.NoError(t, err)
		eInfo, err := expected.FS().Stat("foo")
		require.NoError(t, err)
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = undertest.FS().Chtimes("foo", accessTime, modifyTime)
		assert.NoError(t, err)
		uInfo, err := undertest.FS().Stat("foo")
		require.NoError(t, err)
		undertest.Clean()

		assertEqualFileInfo(t, eInfo, uInfo)
	})
}

// TODO Symlink
