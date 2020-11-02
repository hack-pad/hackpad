// +build js

package interop

import (
	"bytes"
	"context"
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

	StartDownload("application/octet-stream", "go-wasm-mem.pprof", buf.Bytes())
	return nil
}

func StartCPUProfile(ctx context.Context) error {
	var buf bytes.Buffer
	err := pprof.StartCPUProfile(&buf)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		pprof.StopCPUProfile()
		StartDownload("application/octet-stream", "go-wasm-cpu.pprof", buf.Bytes())
	}()
	return nil
}
