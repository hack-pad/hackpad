package fs

import (
	"encoding/base64"
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

func (l *localStorer) GetData(key string) ([]byte, error) {
	defer interop.PanicLogger()
	log.Warn("Getting data ", key, l)
	value := l.getItem.Invoke(localStorageKeyPrefix + key)
	if value.IsNull() {
		log.Warn("No data ", key)
		return nil, os.ErrNotExist
	}
	log.Warn("Got data ", value.Length())
	return base64.StdEncoding.DecodeString(value.String())
}

func (l *localStorer) SetData(key string, data []byte) error {
	defer interop.PanicLogger()
	log.Warn("Setting data ", key, " ", len(data))
	if data == nil {
		l.removeItem.Invoke(localStorageKeyPrefix + key)
		return nil
	}
	l.setItem.Invoke(localStorageKeyPrefix+key, base64.StdEncoding.EncodeToString(data))
	return nil
}
