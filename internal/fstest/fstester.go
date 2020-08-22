package fstest

import (
	"testing"

	"github.com/spf13/afero"
)

type FSTester interface {
	FS() afero.Fs
	WriteFS() afero.Fs
	Clean()
}

type CleanFunc func() error
type CommitWritesFunc func() error

type fsTester struct {
	tb             testing.TB // only used for t.Helper()
	name           string
	fs             afero.Fs
	cleanUp        CleanFunc
	writeFs        afero.Fs
	fetchedWriteFs bool
	commitWrites   CommitWritesFunc
}

func NewTester(tb testing.TB, fs afero.Fs, cleanUp CleanFunc) FSTester {
	return fsTester{
		tb:      tb,
		name:    fs.Name(),
		fs:      fs,
		cleanUp: cleanUp,
	}
}

func (f fsTester) withName(name string) fsTester {
	f.name = name
	return f
}

func (f fsTester) WithFSWriter(fs afero.Fs, commitWrites CommitWritesFunc) fsTester {
	f.writeFs = fs
	f.commitWrites = commitWrites
	return f
}

func (f fsTester) FS() afero.Fs {
	if f.commitWrites != nil && f.fetchedWriteFs {
		// if something was possibly written earlier, then commit those writes
		f.fetchedWriteFs = false
		f.commitWrites()
	}
	return f.fs
}

func (f fsTester) Clean() {
	f.tb.Helper()
	err := f.cleanUp()
	if err != nil {
		f.tb.Errorf("Failed to clean up %s: %v", f.name, err)
	}
}

func (f fsTester) WriteFS() afero.Fs {
	if f.writeFs != nil {
		f.fetchedWriteFs = true // mark a possible write for later commit
		return f.writeFs
	}
	return f.fs
}
