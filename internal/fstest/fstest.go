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
	defer os.Chdir(wd)

	RunWith(t, fs, afero.NewOsFs(), cleanUp, cleanUpOsFs)
}

// RunWith runs filesystem tests on 'undertest' and compares with the expected results of 'expected'.
// The cleanFunc's are run after every subtest and once before the first test.
func RunWith(
	t *testing.T,
	undertest, expected afero.Fs,
	cleanTest, cleanExpected CleanFunc,
) {
	t.Helper()

	fsTest := fsTest{
		T:         t,
		undertest: newTester(t, "undertest", undertest, cleanTest),
		expected:  newTester(t, "expected", expected, cleanExpected),
	}
	fsTest.Clean()

	// Establish baseline tests for things like Stat and Readdirnames.
	// With basic forms of these passing, we can reuse them in subsequent tests without worrying about confusing overlap.
	for _, result := range []bool{
		fsTest.Run("basic fs.Create", TestFsBasicCreate),
		fsTest.Run("basic fs.Mkdir", TestFsBasicMkdir),
		fsTest.Run("fs.Open", TestFsOpen),
		fsTest.Run("fs.Stat", TestFsStat),
		fsTest.Run("file.Readdirnames", TestFileReaddirnames),
	} {
		if !result {
			t.Fatal("Cannot verify further tests without basic scenarios passing. (e.g. fs.Stat and file.Readdirnames)")
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
}

// cleanUpOsFs runs basic cleanup for the current directory.
// Run automatically creates and uses a temp dir, so this cleans its contents.
func cleanUpOsFs() error {
	file, err := os.Open(".")
	if err != nil {
		return err
	}
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
