package interop

import (
	"bytes"
	"runtime"
	"runtime/pprof"
	"syscall/js"
)

func MemoryProfile(this js.Value, args []js.Value) interface{} {
	var buf bytes.Buffer
	runtime.GC()
	err := pprof.WriteHeapProfile(&buf)
	if err != nil {
		return err
	}

	StartDownload("application/octet-stream", "go-wasm.pprof", buf.Bytes())
	return nil
}
