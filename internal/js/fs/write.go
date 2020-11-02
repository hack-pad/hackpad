// +build js

package fs

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/process"
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
	jsBuffer := args[1]
	offset := 0
	if len(args) >= 3 {
		offset = args[2].Int()
	}
	length := jsBuffer.Length()
	if len(args) >= 4 {
		length = args[3].Int()
	}
	var position *int64
	if len(args) >= 5 && args[4].Type() == js.TypeNumber {
		position = new(int64)
		*position = int64(args[4].Int())
	}

	buffer := make([]byte, length)
	js.CopyBytesToGo(buffer, jsBuffer)
	p := process.Current()
	n, err := p.Files().Write(fd, buffer, offset, length, position)
	js.CopyBytesToJS(jsBuffer, buffer)
	return []interface{}{n, jsBuffer}, err
}
