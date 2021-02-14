// +build js

package interop

import (
	"bytes"
	"context"
	"runtime"
	"runtime/pprof"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/log"
)

func ProfileJS(this js.Value, args []js.Value) interface{} {
	MemoryProfileJS(this, args)
	//StartCPUProfileJS(this, args) // Re-enable once CPU profiles actually work in the browser. Currently produces 0 samples.
	return nil
}

func MemoryProfile() ([]byte, error) {
	var buf bytes.Buffer
	runtime.GC()
	err := pprof.WriteHeapProfile(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MemoryProfileJS(this js.Value, args []js.Value) interface{} {
	buf, err := MemoryProfile()
	if err != nil {
		log.Error("Failed to create memory profile: ", err)
		return nil
	}
	StartDownload("application/octet-stream", "go-wasm-mem.pprof", buf)
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

func StartCPUProfileDuration(d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	go func() {
		// NOTE: this is purely to satisfy linters. This func is only called while debugging. We don't want to cancel the context in this scope, so discard it.
		<-ctx.Done()
		cancel()
	}()
	return StartCPUProfile(ctx)
}

func StartCPUProfileJS(this js.Value, args []js.Value) interface{} {
	duration := 30 * time.Second
	if len(args) > 0 && args[0].Truthy() {
		duration = time.Duration(args[0].Float() * float64(time.Second))
	}
	err := StartCPUProfileDuration(duration)
	if err != nil {
		log.Error("Failed to start CPU profile: ", err)
	}
	return nil
}
