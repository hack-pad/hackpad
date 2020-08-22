package fstest

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
)

// Run runs filesystem tests and compares results with an *afero.OsFs.
// 'cleanUp' is run after every subtest and once before the first test.
func Run(t *testing.T, fs afero.Fs, cleanUp CleanFunc) {
	t.Helper()

	prepOsFsSuite(t) // It's the caller's responsibility to handle setup for fs.

	undertest := NewTester(t, fs, cleanUp)
	expected := NewTester(t, afero.NewOsFs(), cleanUpOsFs)
	runWith(t, undertest, expected)
}

func prepOsFsSuite(t *testing.T) {
	// To prepare an OsFs, chdir to a temp dir sandbox and disable umask.
	oldmask := setUmask(0)
	t.Cleanup(func() {
		setUmask(oldmask)
	})

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal("Failed to setup temporary directory for an OsFs:", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
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
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
}

// runWith runs filesystem tests on 'undertest' and compares with the expected results of 'expected'.
// The cleanFunc's are run after every subtest and once before the first test.
func runWith(
	t *testing.T,
	undertest, expected FSTester,
) {
	t.Helper()

	if tester, ok := undertest.(*fsTester); ok {
		undertest = tester.withName("undertest")
	}
	if tester, ok := expected.(*fsTester); ok {
		expected = tester.withName("expected")
	}

	fsTest := fsTest{
		T:         t,
		undertest: undertest,
		expected:  expected,
	}
	fsTest.Clean()

	// Establish baseline tests for things like Stat and Readdirnames.
	// With basic forms of these passing, we can reuse them in subsequent tests without worrying about confusing overlap.
	for _, result := range []bool{
		fsTest.Run("basic fs.Create", TestFsBasicCreate),
		fsTest.Run("basic fs.Mkdir", TestFsBasicMkdir),
		fsTest.Run("basic fs.Chmod", TestFsBasicChmod),
		fsTest.Run("basic fs.Chtimes", TestFsBasicChtimes),
		fsTest.Run("fs.Open", TestFsOpen),
		fsTest.Run("fs.Stat", TestFsStat),
		fsTest.Run("file.Readdirnames", TestFileReaddirnames),
		fsTest.Run("file.Close", TestFileClose),
	} {
		if !result {
			t.Skip("Cannot verify further tests without basic scenarios passing. (e.g. fs.Stat and file.Readdirnames)")
		}
	}

	fsTest.Run("fs.Create", TestFsCreate)
	fsTest.Run("fs.Mkdir", TestFsMkdir)
	fsTest.Run("fs.MkdirAll", TestFsMkdirAll)
	fsTest.Run("fs.OpenFile", TestFsOpenFile)
	fsTest.Run("fs.Remove", TestFsRemove)
	fsTest.Run("fs.RemoveAll", TestFsRemoveAll)
	fsTest.Run("fs.Rename", TestFsRename)
	fsTest.Run("fs.Chmod", TestFsChmod)
	fsTest.Run("fs.Chtimes", TestFsChtimes)
	// TODO fs.Symlink

	fsTest.Run("file.Read", TestFileRead)
	fsTest.Run("file.ReadAt", TestFileReadAt)
	fsTest.Run("file.Seek", TestFileSeek)
	fsTest.Run("file.Write", TestFileWrite)
	fsTest.Run("file.WriteAt", TestFileWriteAt)
	fsTest.Run("file.Readdir", TestFileReaddir)
	fsTest.Run("file.Stat", TestFileStat)
	fsTest.Run("file.Sync", TestFileSync)
	fsTest.Run("file.Truncate", TestFileTruncate)
	fsTest.Run("file.WriteString", TestFileWriteString)
}

// cleanUpOsFs runs basic cleanup for the current directory.
// Run automatically creates and uses a temp dir, so this cleans its contents.
func cleanUpOsFs() error {
	file, err := os.Open(".")
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	names, err := file.Readdirnames(0)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := os.RemoveAll(name); err != nil {
			return err
		}
	}
	return nil
}

type fsTestFunc func(t *testing.T, undertest, expected FSTester)

type fsTest struct {
	T                   *testing.T
	undertest, expected FSTester
}

func (f *fsTest) Clean() {
	f.T.Helper()
	f.expected.Clean()
	f.undertest.Clean()
}

// Run is a convenience func for running a subtest
func (f *fsTest) Run(name string, test fsTestFunc) bool {
	f.T.Helper()
	return f.T.Run(name, func(t *testing.T) {
		t.Helper()
		defer f.Clean()

		test(t, f.undertest, f.expected)
	})
}
