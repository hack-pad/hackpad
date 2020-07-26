package fs

import (
	"encoding/json"
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/storer"
	"github.com/johnstarich/go-wasm/log"
)

const (
	localStorageKeyPrefix = "storer$1$"
)

// TODO consider adding "locks": https://balpha.de/2012/03/javascript-concurrency-and-locking-the-html5-localstorage/

type LocalStorageFs struct {
	*storer.Fs
}

func NewJSStorage(s js.Value) *LocalStorageFs {
	fs := storer.New(&localStorer{
		getItem:    s.Get("getItem"),
		setItem:    s.Get("setItem"),
		removeItem: s.Get("removeItem"),
	})
	return &LocalStorageFs{fs}
}

type localStorer struct {
	getItem, setItem, removeItem js.Value
}

func (l *localStorer) GetFileRecord(path string, dest *storer.FileRecord) error {
	defer interop.PanicLogger()
	log.Warn("Getting data ", path, l)
	value := l.getItem.Invoke(localStorageKeyPrefix + path)
	if value.IsNull() {
		log.Warn("No data ", path)
		return os.ErrNotExist
	}
	log.Warn("Got data ", value.Length())
	return json.Unmarshal([]byte(value.String()), dest)
}

func (l *localStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	defer interop.PanicLogger()
	log.Warn("Setting data ", path)
	if data == nil {
		l.removeItem.Invoke(localStorageKeyPrefix + path)
		return nil
	}
	buf, err := json.Marshal(data)
	if err == nil {
		l.setItem.Invoke(localStorageKeyPrefix+path, string(buf))
	}
	return err
}
