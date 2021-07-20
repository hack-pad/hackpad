package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	goPath "path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/johnstarich/go-wasm/internal/fstest"
	"github.com/johnstarich/go-wasm/internal/fsutil"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type swappableFS struct {
	afero.Fs
}

func newSwappableFS(fs afero.Fs) *swappableFS {
	return &swappableFS{fs}
}

func (s *swappableFS) Swap(fs afero.Fs) {
	s.Fs = fs
}

func TestFs(t *testing.T) {
	tarFS := newSwappableFS(nil)
	memFS := newSwappableFS(afero.NewMemMapFs())

	rebuildTarFromMem := func() error {
		tmpTarFS, err := newTarFromFS(memFS)
		if err == nil {
			tarFS.Swap(tmpTarFS)
		}
		return err
	}
	require.NoError(t, rebuildTarFromMem())

	cleanup := func() error {
		memFS.Swap(afero.NewMemMapFs())
		err := rebuildTarFromMem()
		return err
	}

	fstest.RunReadOnly(t, tarFS, memFS, cleanup, rebuildTarFromMem)
}

func newTarFromFS(src afero.Fs) (*FS, error) {
	r, err := buildTarFromFS(src)
	if err != nil {
		return nil, err
	}
	return New(r, afero.NewMemMapFs())
}

func buildTarFromFS(src afero.Fs) (io.Reader, error) {
	var buf bytes.Buffer
	compressor := gzip.NewWriter(&buf)
	defer compressor.Close()

	archive := tar.NewWriter(compressor)
	defer archive.Close()

	err := afero.Walk(src, "/", copyTarWalk(src, archive))
	return &buf, errors.Wrap(err, "Failed building tar from FS walk")
}

func copyTarWalk(src afero.Fs, archive *tar.Writer) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path
		if info.IsDir() {
			header.Name += "/"
		}
		err = archive.WriteHeader(header)
		if err != nil {
			return err
		}
		fileBytes, err := afero.ReadFile(src, path)
		if err != nil {
			return err
		}
		_, err = archive.Write(fileBytes)
		return err
	}
}

func TestNewFromFs(t *testing.T) {
	for _, tc := range []struct {
		description string
		do          func(t *testing.T, fs afero.Fs)
	}{
		{
			description: "empty",
			do:          func(t *testing.T, fs afero.Fs) {},
		},
		{
			description: "one file",
			do: func(t *testing.T, fs afero.Fs) {
				_, err := fs.Create("foo")
				require.NoError(t, err)
			},
		},
		{
			description: "one dir",
			do: func(t *testing.T, fs afero.Fs) {
				err := fs.Mkdir("foo", 0700)
				require.NoError(t, err)
			},
		},
		{
			description: "dir with one nested file",
			do: func(t *testing.T, fs afero.Fs) {
				err := fs.Mkdir("foo", 0700)
				require.NoError(t, err)
				_, err = fs.Create("foo/bar")
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			memFS := afero.NewMemMapFs()
			tc.do(t, memFS)
			timer := time.NewTimer(50 * time.Millisecond)
			done := make(chan struct{})

			go func() {
				tarFS, err := newTarFromFS(memFS)
				assert.NoError(t, err)
				assert.NotNil(t, tarFS)
				close(done)
			}()

			select {
			case <-done:
				timer.Stop()
			case <-timer.C:
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, true)
				t.Fatalf("Took too long:\n%s", string(buf[:n]))
			}
		})
	}
}

func TestDirsFromPath(t *testing.T) {
	for _, tc := range []struct {
		description string
		path        string
		expect      []string
	}{
		{
			description: "empty path",
			path:        "",
			expect:      []string{"/"},
		},
		{
			description: "slash",
			path:        "/",
			expect:      []string{"/"},
		},
		{
			description: "file",
			path:        "/foo",
			expect:      []string{"/"},
		},
		{
			description: "dir",
			path:        "/foo/",
			expect:      []string{"/foo", "/"},
		},
		{
			description: "nested dir",
			path:        "/foo/bar/",
			expect:      []string{"/foo/bar", "/foo", "/"},
		},
		{
			description: "nested file",
			path:        "/foo/bar",
			expect:      []string{"/foo", "/"},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expect, dirsFromPath(tc.path))
		})
	}
}

// dirsFromPath returns all directory segments in 'path'. Assumes 'path' is a raw header name from a tar.
func dirsFromPath(path string) []string {
	var dirs []string
	if strings.HasSuffix(path, pathSeparator) {
		// denotes a tar directory path, so clean it and add it before pop
		path = fsutil.NormalizePath(path)
		dirs = append(dirs, path)
	}
	if path == pathSeparator {
		return dirs
	}
	path = fsutil.NormalizePath(path) // make absolute + clean
	path = goPath.Dir(path)           // pop normal files from the end
	var prevPath string
	for ; path != prevPath; path = goPath.Dir(path) {
		dirs = append(dirs, path)
		prevPath = path
	}
	return dirs
}
