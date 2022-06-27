package worker

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
)

const (
	ofFilePath   = "filePath"
	ofSeekOffset = "seekOffset"
	ofPipe       = "pipe"
)

type openFile struct {
	filePath   string
	seekOffset int64
	pipe       *jsworker.MessagePort
}

func readOpenFile(v js.Value) openFile {
	props := interop.Entries(v)
	return openFile{
		filePath:   optionalString(props[ofFilePath]),
		seekOffset: optionalInt64(props[ofSeekOffset]),
		pipe:       optionalPipe(props[ofPipe]),
	}
}

func (o openFile) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		ofFilePath:   o.filePath,
		ofSeekOffset: o.seekOffset,
		ofPipe:       o.pipe.JSValue(),
	})
}

func optionalString(v js.Value) string {
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}

func optionalInt64(v js.Value) int64 {
	if v.Type() != js.TypeNumber {
		return 0
	}
	return int64(v.Int())
}
