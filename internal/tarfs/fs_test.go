package tarfs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/fstest"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS(t *testing.T) {
	options := fstest.FSOptions{
		Name: "tar",
		Setup: fstest.TestSetupFunc(func(tb testing.TB) (fstest.SetupFS, func() hackpadfs.FS) {
			setupFS, err := mem.NewFS()
			require.NoError(tb, err)

			return setupFS, func() hackpadfs.FS {
				return newTarFromFS(tb, setupFS)
			}
		}),
	}
	fstest.FS(t, options)
	fstest.File(t, options)
}

func newTarFromFS(tb testing.TB, src hackpadfs.FS) *FS {
	r, err := buildTarFromFS(src)
	require.NoError(tb, err)
	memFS, err := mem.NewFS()
	require.NoError(tb, err)

	fs, err := New(r, memFS)
	require.NoError(tb, err)
	return fs
}

func buildTarFromFS(src hackpadfs.FS) (io.Reader, error) {
	var buf bytes.Buffer
	compressor := gzip.NewWriter(&buf)
	defer compressor.Close()

	archive := tar.NewWriter(compressor)
	defer archive.Close()

	err := hackpadfs.WalkDir(src, ".", copyTarWalk(src, archive))
	return &buf, errors.Wrap(err, "Failed building tar from FS walk")
}

func copyTarWalk(src hackpadfs.FS, archive *tar.Writer) hackpadfs.WalkDirFunc {
	return func(path string, dir hackpadfs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := dir.Info()
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
		fileBytes, err := hackpadfs.ReadFile(src, path)
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
		do          func(t *testing.T, fs hackpadfs.FS)
	}{
		{
			description: "empty",
			do:          func(t *testing.T, fs hackpadfs.FS) {},
		},
		{
			description: "one file",
			do: func(t *testing.T, fs hackpadfs.FS) {
				_, err := hackpadfs.Create(fs, "foo")
				require.NoError(t, err)
			},
		},
		{
			description: "one dir",
			do: func(t *testing.T, fs hackpadfs.FS) {
				err := hackpadfs.Mkdir(fs, "foo", 0700)
				require.NoError(t, err)
			},
		},
		{
			description: "dir with one nested file",
			do: func(t *testing.T, fs hackpadfs.FS) {
				err := hackpadfs.Mkdir(fs, "foo", 0700)
				require.NoError(t, err)
				_, err = hackpadfs.Create(fs, "foo/bar")
				require.NoError(t, err)
			},
		},
	} {
		t.Run(tc.description, func(t *testing.T) {
			memFS, err := mem.NewFS()
			require.NoError(t, err)
			tc.do(t, memFS)
			timer := time.NewTimer(50 * time.Millisecond)
			done := make(chan struct{})

			go func() {
				tarFS := newTarFromFS(t, memFS)
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
