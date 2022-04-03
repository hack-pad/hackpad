// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/pkg/errors"
)

func (s fileShim) read(args []js.Value) ([]interface{}, error) {
	n, buf, err := s.readSyncImpl(args)
	return []interface{}{n, buf}, err
}

func (s fileShim) readSync(args []js.Value) (interface{}, error) {
	n, _, err := s.readSyncImpl(args)
	return n, err
}

func (s fileShim) readSyncImpl(args []js.Value) (int, js.Value, error) {
	// args: fd, buffer, offset, length, position
	if len(args) != 5 {
		return 0, js.Null(), errors.Errorf("missing required args, expected 5: %+v", args)
	}
	fd := fs.FID(args[0].Int())
	buffer, err := idbblob.New(args[1])
	if err != nil {
		return 0, js.Null(), err
	}
	offset := args[2].Int()
	length := args[3].Int()
	var position *int64
	if args[4].Type() == js.TypeNumber {
		position = new(int64)
		*position = int64(args[4].Int())
	}

	n, err := s.process.Files().Read(fd, buffer, offset, length, position)
	return n, buffer.JSValue(), err
}
