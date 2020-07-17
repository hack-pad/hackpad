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

var jsUint8Array = js.Global().Get("Uint8Array")

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

func (l *localStorer) GetFileRecord(key string, dest *storer.FileRecord) error {
	defer interop.PanicLogger()
	log.Warn("Getting data ", key, l)
	value := l.getItem.Invoke(localStorageKeyPrefix + key)
	if value.IsNull() {
		log.Warn("No data ", key)
		return os.ErrNotExist
	}
	log.Warn("Got data ", value.Length())
	return json.Unmarshal([]byte(value.String()), dest)
}

func (l *localStorer) SetFileRecord(key string, data *storer.FileRecord) error {
	defer interop.PanicLogger()
	log.Warn("Setting data ", key)
	if data == nil {
		l.removeItem.Invoke(localStorageKeyPrefix + key)
		return nil
	}
	buf, err := json.Marshal(data)
	if err == nil {
		l.setItem.Invoke(localStorageKeyPrefix+key, string(buf))
	}
	return err
}
