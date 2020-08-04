package fstest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fsStat interface {
	Info() os.FileInfo
	ReadDir() []fsStat
	String() string
}

type fsStatFile struct {
	info  os.FileInfo
	files []fsStat
}

func (f *fsStatFile) Info() os.FileInfo {
	return f.info
}

func (f *fsStatFile) ReadDir() []fsStat {
	return f.files
}

func (f *fsStatFile) stringIndent(indent int) string {
	var s strings.Builder
	s.WriteString(f.info.Name())
	if f.info.IsDir() {
		s.WriteRune('/')
	} else {
		s.WriteString(fmt.Sprintf(" (%d bytes)", f.info.Size()))
	}
	s.WriteRune('\n')
	indentStr := fmt.Sprintf(fmt.Sprintf("\t%%%ds", indent*4), " ")
	for _, file := range f.files {
		s.WriteString(indentStr)
		s.WriteString(file.(*fsStatFile).stringIndent(indent + 1))
		s.WriteRune('\n')
	}
	return s.String()
}

func (f *fsStatFile) String() string {
	return f.stringIndent(0)
}

func statFS(t *testing.T, fs afero.Fs) fsStat {
	return statFSPath(t, fs, ".")
}

func statFSPath(t *testing.T, fs afero.Fs, path string) fsStat {
	stat := &fsStatFile{}
	var err error
	stat.info, err = fs.Stat(path)
	require.NoError(t, err)
	if !stat.info.IsDir() {
		return stat
	}

	file, err := fs.Open(path)
	require.NoError(t, err)
	names, err := file.Readdirnames(0)
	require.NoError(t, err)
	stat.files = make([]fsStat, len(names))

	for i := range names {
		stat.files[i] = statFSPath(t, fs, filepath.Join(path, names[i]))
	}
	return stat
}

func assertEqualFS(t *testing.T, expected, actual fsStat) {
	t.Helper()
	assertEqualFileInfo(t, expected.Info(), actual.Info())
	expectedDir := expected.ReadDir()
	actualDir := actual.ReadDir()

	if len(expectedDir) != len(actualDir) {
		expectedNames := make([]string, len(expectedDir))
		for i := range expectedDir {
			expectedNames[i] = expectedDir[i].Info().Name()
		}
		actualNames := make([]string, len(actualDir))
		for i := range actualDir {
			actualNames[i] = actualDir[i].Info().Name()
		}
		require.Equal(t, expectedNames, actualNames)
	}

	for i := range expectedDir {
		assertEqualFS(t, expectedDir[i], actualDir[i])
	}
}

func assertEqualFileInfo(t *testing.T, expected, actual os.FileInfo) {
	t.Helper()
	type info struct {
		Name  string
		Size  int64
		Mode  os.FileMode
		IsDir bool
	}
	expectedInfo := info{
		Name:  expected.Name(),
		Size:  expected.Size(),
		Mode:  expected.Mode(),
		IsDir: expected.IsDir(),
	}
	actualInfo := info{
		Name:  actual.Name(),
		Size:  actual.Size(),
		Mode:  actual.Mode(),
		IsDir: actual.IsDir(),
	}
	expectedModTime := expected.ModTime()
	actualModTime := actual.ModTime()
	if expectedInfo.Name == "." && actualInfo.Name == "" {
		// not all file systems are consistent when running Stat(".")
		actualInfo.Name = expectedInfo.Name
		actualModTime = expectedModTime
	}
	if actualInfo.IsDir {
		actualInfo.Size = expectedInfo.Size // size doesn't matter for directories
	}
	assert.Equalf(t, expectedInfo, actualInfo, "FileInfo differs for file %q", expectedInfo.Name) // use structs for compact comparison output
	assert.WithinDurationf(t, expectedModTime, actualModTime, 10*time.Second, "Mod time for %q not close enough", expectedInfo.Name)
}
