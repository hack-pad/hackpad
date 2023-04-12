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

func writeSync(args []js.Value) (interface{}, error) {
	ret, err := write(args)
	if len(ret) > 1 {
		return ret[0], err
	}
	return ret, err
}

func write(args []js.Value) ([]interface{}, error) {
	// args: fd, buffer, offset, length, position
	if len(args) < 2 {
		return nil, errors.Errorf("missing required args, expected fd and buffer: %+v", args)
	}
	fd := fs.FID(args[0].Int())
	buffer, err := idbblob.New(args[1])
	if err != nil {
		return nil, err
	}
	offset := 0
	if len(args) >= 3 {
		offset = args[2].Int()
	}
	length := buffer.Len()
	if len(args) >= 4 {
		length = args[3].Int()
	}
	var position *int64
	if len(args) >= 5 && args[4].Type() == js.TypeNumber {
		position = new(int64)
		*position = int64(args[4].Int())
	}

	p := process.Current()
	n, err := p.Files().Write(fd, buffer, offset, length, position)
	return []interface{}{n, buffer.JSValue()}, err
}
