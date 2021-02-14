package fs

import (
	"time"

	"github.com/johnstarich/go-wasm/log"
	"go.uber.org/atomic"
)

type workingDirectory struct {
	path     atomic.String
	updating atomic.Bool
}

func newWorkingDirectory(path string) *workingDirectory {
	w := &workingDirectory{}
	w.path.Store(path)
	return w
}

func (w *workingDirectory) Set(wd string) error {
	// must be async to support IDB FS
	w.updating.Store(true)
	go func() {
		defer w.updating.Store(false)
		info, err := filesystem.Stat(wd)
		if err != nil {
			log.Error("Cannot chdir to ", wd, ": ", err)
			return
		}
		if !info.IsDir() {
			log.Error("Cannot chdir to ", wd, ": ", ErrNotDir)
			return
		}
		w.path.Store(wd)
	}()
	return nil
}

func (w *workingDirectory) Get() (string, error) {
	for i := 0; i < 10 && w.updating.Load(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	return w.path.Load(), nil
}
