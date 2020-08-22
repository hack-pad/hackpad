package fstest

import (
	"testing"

	"github.com/spf13/afero"
)

// RunReadOnly runs read-only filesystem tests on 'fs' and compares with the expected results of an afero.OsFs.
// The cleanUp is run after every subtest and once before the first test.
func RunReadOnly(t *testing.T, fs, writeFs afero.Fs, cleanUp CleanFunc, commitWrites CommitWritesFunc) {
	t.Helper()

	prepOsFsSuite(t) // It's the caller's responsibility to handle setup for fs.

	undertest := NewTester(t, fs, cleanUp).(fsTester).withName("undertest").WithFSWriter(writeFs, commitWrites)
	expected := NewTester(t, afero.NewOsFs(), cleanUpOsFs).(fsTester).withName("expected")
	fsTest := fsTest{
		T:         t,
		undertest: undertest,
		expected:  expected,
	}
	fsTest.Clean()

	// Establish baseline tests for things like Stat and Readdirnames.
	// With basic forms of these passing, we can reuse them in subsequent tests without worrying about confusing overlap.
	for _, result := range []bool{
		fsTest.Run("fs.Open", TestFsOpen),
		fsTest.Run("fs.Stat", TestFsStat),
		fsTest.Run("file.Readdirnames", TestFileReaddirnames),
		fsTest.Run("file.Close", TestFileClose),
	} {
		if !result {
			t.Skip("Cannot verify further tests without basic scenarios passing. (e.g. fs.Stat and file.Readdirnames)")
		}
	}

	fsTest.Run("file.Read", TestFileRead)
	fsTest.Run("file.ReadAt", TestFileReadAt)
	fsTest.Run("file.Seek", TestFileSeek)
	fsTest.Run("file.Readdir", TestFileReaddir)
	fsTest.Run("file.Stat", TestFileStat)
	fsTest.Run("file.Sync", TestFileSync)
}
