//go:build js
// +build js

package fs

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
	"github.com/pkg/errors"
)

func read(args []js.Value) ([]interface{}, error) {
	n, buf, err := readSyncImpl(args)
	return []interface{}{n, buf}, err
}

func readSync(args []js.Value) (interface{}, error) {
	n, _, err := readSyncImpl(args)
	return n, err
}

func readSyncImpl(args []js.Value) (int, js.Value, error) {
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

	p := process.Current()
	n, err := p.Files().Read(fd, buffer, offset, length, position)
	return n, buffer.JSValue(), err
}
