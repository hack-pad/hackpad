package fstest

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
)

// RunReadOnly runs read-only filesystem tests on 'fs' and compares with the expected results of an afero.OsFs.
// The cleanUp is run after every subtest and once before the first test.
func RunReadOnly(t *testing.T, fs, writeFs afero.Fs, cleanUp CleanFunc) {
	t.Helper()

	// Since expected is an OsFs, chdir to a temp dir sandbox and disable umask.
	// It's the caller's responsibility to handle setup for undertest.
	oldmask := setUmask(0)
	defer setUmask(oldmask)

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("Failed to setup temporary directory for an OsFs:", err)
	}
	defer os.RemoveAll(dir)
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal("Failed to chmod temporary directory:", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current working directory:", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal("Failed to chdir to temporary directory for an OsFs:", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	t.Helper()

	undertest := newTester(t, "undertest", fs, cleanUp)
	undertest.(*fsTester).setFsWriter(writeFs)
	fsTest := fsTest{
		T:         t,
		undertest: undertest,
		expected:  newTester(t, "expected", afero.NewOsFs(), cleanUpOsFs),
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
