package main

import (
	"bytes"
	"runtime"
	"runtime/pprof"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
)

func profileFunc(this js.Value, args []js.Value) interface{} {
	var buf bytes.Buffer
	runtime.GC()
	err := pprof.WriteHeapProfile(&buf)
	if err != nil {
		return err
	}

	interop.StartDownload("application/octet-stream", "go-wasm.pprof", buf.Bytes())
	return nil
}
