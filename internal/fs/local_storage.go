// +build js

package fs

import (
	"encoding/json"
	"os"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/blob"
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

type jsonFileRecord struct {
	Data     []byte
	DirNames []string
	ModTime  time.Time
	Mode     os.FileMode
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
	var jDest jsonFileRecord
	err := json.Unmarshal([]byte(value.String()), &jDest)

	dest.DataFn = func() (blob.Blob, error) {
		return blob.NewFromBytes(jDest.Data), nil
	}
	dest.InitialSize = int64(len(jDest.Data))
	dest.DirNamesFn = func() ([]string, error) {
		return jDest.DirNames, nil
	}
	dest.ModTime = jDest.ModTime
	dest.Mode = jDest.Mode
	return err
}

func (l *localStorer) SetFileRecord(path string, data *storer.FileRecord) error {
	defer interop.PanicLogger()
	log.Warn("Setting data ", path)
	if data == nil {
		l.removeItem.Invoke(localStorageKeyPrefix + path)
		return nil
	}
	jFileRecord := jsonFileRecord{
		Data:     data.Data().Bytes(),
		DirNames: data.DirNames(),
		ModTime:  data.ModTime,
		Mode:     data.Mode,
	}
	buf, err := json.Marshal(jFileRecord)
	if err == nil {
		l.setItem.Invoke(localStorageKeyPrefix+path, string(buf))
	}
	return err
}
