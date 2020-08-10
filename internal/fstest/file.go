package fstest

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileClose(t *testing.T, undertest, expected FSTester) {
	f, err := expected.FS().Create("foo")
	require.NoError(t, err)
	assert.NoError(t, f.Close())
	assert.Error(t, f.Close())
	expected.Clean()

	f, err = undertest.FS().Create("foo")
	require.NoError(t, err)
	assert.NoError(t, f.Close())
	assert.Error(t, f.Close())
	undertest.Clean()
}

func TestFileRead(t *testing.T, undertest, expected FSTester) {
	const fileContents = "hello world"
	t.Run("read empty", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		buf := make([]byte, 10)
		n, err := f.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		buf = make([]byte, 10)
		n, err = f.Read(buf)
		assert.Equal(t, 0, n)
		assert.Equal(t, io.EOF, err)
		require.NoError(t, f.Close())
		undertest.Clean()
	})

	t.Run("read a few bytes at a time", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		f, err = expected.FS().Open("foo")
		require.NoError(t, err)

		const firstBufLen = 2
		buf := make([]byte, firstBufLen)
		n, err := f.Read(buf)
		assert.Equal(t, firstBufLen, n)
		assert.NoError(t, err)
		assert.Equal(t, "he", string(buf))

		buf = make([]byte, len(fileContents)*2)
		n, err = f.Read(buf)
		assert.Equal(t, len(fileContents)-firstBufLen, n)
		if err == nil {
			// it's ok to return a nil error when finishing a read
			// but the next read must return 0 and EOF
			tmpBuf := make([]byte, len(buf))
			var zeroN int
			zeroN, err = f.Read(tmpBuf)
			assert.Equal(t, 0, zeroN)
		}
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, "llo world", string(buf[:n]))
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		f, err = undertest.FS().Open("foo")
		require.NoError(t, err)

		buf = make([]byte, firstBufLen)
		n, err = f.Read(buf)
		assert.Equal(t, firstBufLen, n)
		assert.NoError(t, err)
		assert.Equal(t, "he", string(buf))

		buf = make([]byte, len(fileContents)*2)
		n, err = f.Read(buf)
		assert.Equal(t, len(fileContents)-firstBufLen, n)
		if err == nil {
			// it's ok to return a nil error when finishing a read
			// but the next read must return 0 and EOF
			tmpBuf := make([]byte, len(buf))
			var zeroN int
			zeroN, err = f.Read(tmpBuf)
			assert.Equal(t, 0, zeroN)
		}
		assert.Equal(t, io.EOF, err)
		assert.Equal(t, "llo world", string(buf[:n]))
		require.NoError(t, f.Close())
		undertest.Clean()
	})
}

func TestFileReadAt(t *testing.T, undertest, expected FSTester) {
	for _, fsTest := range []FSTester{expected, undertest} {
		const fileContents = "hello world"
		f, err := fsTest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)

		for _, tc := range []struct {
			description string
			bufSize     int
			off         int64
			expectN     int
			expectBuf   string
			expectErr   error
		}{
			{
				description: "at start",
				bufSize:     len(fileContents),
				off:         0,
				expectN:     len(fileContents),
				expectBuf:   "hello world",
			},
			{
				description: "negative offset",
				bufSize:     len(fileContents),
				off:         -1,
				expectErr:   errors.New("negative offset"),
			},
			{
				description: "small byte offset",
				bufSize:     len(fileContents),
				off:         2,
				expectN:     len(fileContents) - 2,
				expectBuf:   "llo world",
				expectErr:   io.EOF,
			},
			{
				description: "small read at offset",
				bufSize:     2,
				off:         2,
				expectN:     2,
				expectBuf:   "ll",
			},
			{
				description: "full read at offset",
				bufSize:     len(fileContents),
				off:         2,
				expectN:     len(fileContents) - 2,
				expectBuf:   "llo world",
				expectErr:   io.EOF,
			},
		} {
			t.Run(tc.description, func(t *testing.T) {
				buf := make([]byte, tc.bufSize)
				n, err := f.ReadAt(buf, tc.off)
				if n == tc.bufSize && err == io.EOF {
					err = nil
				}
				if tc.expectErr != nil {
					if pathErr, ok := err.(*os.PathError); ok {
						assert.Equal(t, "readat", pathErr.Op)
						assert.Equal(t, "foo", strings.TrimPrefix(pathErr.Path, "/"))
						err = pathErr.Err
					}
					assert.Equal(t, tc.expectErr, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tc.expectN, n)
				assert.Equal(t, tc.expectBuf, string(buf[:n]))
			})
		}

		require.NoError(t, f.Close())
		fsTest.Clean()
	}
}

func TestFileSeek(t *testing.T, undertest, expected FSTester) {
	const fileContents = "hello world"

	t.Run("seek start", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		const offset = 1
		off, err := f.Seek(offset, io.SeekStart)
		assert.NoError(t, err)
		assert.EqualValues(t, offset, off)
		buf := make([]byte, len(fileContents))
		n, err := f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "ello world", string(buf[:n]))
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		off, err = f.Seek(offset, io.SeekStart)
		assert.NoError(t, err)
		assert.EqualValues(t, offset, off)
		buf = make([]byte, len(fileContents))
		n, err = f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "ello world", string(buf[:n]))
		require.NoError(t, f.Close())
		undertest.Clean()
	})

	t.Run("seek current", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		const firstSeekOff = 5
		const offset = -1
		_, err = f.Seek(firstSeekOff, io.SeekStart) // get close to middle
		require.NoError(t, err)
		off, err := f.Seek(offset, io.SeekCurrent)
		assert.NoError(t, err)
		assert.EqualValues(t, firstSeekOff+offset, off)
		buf := make([]byte, len(fileContents))
		n, err := f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "o world", string(buf[:n]))
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		_, err = f.Seek(firstSeekOff, io.SeekStart) // get close to middle
		require.NoError(t, err)
		off, err = f.Seek(offset, io.SeekCurrent)
		assert.NoError(t, err)
		assert.EqualValues(t, firstSeekOff+offset, off)
		buf = make([]byte, len(fileContents))
		n, err = f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "o world", string(buf[:n]))
		require.NoError(t, f.Close())
		undertest.Clean()
	})

	t.Run("seek end", func(t *testing.T) {
		f, err := expected.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		const offset = -1
		off, err := f.Seek(offset, io.SeekEnd)
		assert.NoError(t, err)
		assert.EqualValues(t, len(fileContents)+offset, off)
		buf := make([]byte, len(fileContents))
		n, err := f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "d", string(buf[:n]))
		require.NoError(t, f.Close())
		expected.Clean()

		f, err = undertest.FS().Create("foo")
		require.NoError(t, err)
		_, err = f.Write([]byte(fileContents))
		require.NoError(t, err)
		off, err = f.Seek(offset, io.SeekEnd)
		assert.NoError(t, err)
		assert.EqualValues(t, len(fileContents)+offset, off)
		buf = make([]byte, len(fileContents))
		n, err = f.Read(buf)
		require.True(t, err == nil || err == io.EOF)
		assert.Equal(t, "d", string(buf[:n]))
		require.NoError(t, f.Close())
		undertest.Clean()
	})
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
