//go:build js
// +build js

package interop

import (
	"bytes"
	"context"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"syscall/js"
	"time"

	"github.com/hack-pad/hackpad/internal/log"
)

func ProfileJS(this js.Value, args []js.Value) interface{} {
	go func() {
		MemoryProfileJS(this, args)
		// Re-enable once these profiles actually work in the browser. Currently produces 0 samples.
		// TraceProfileJS(this, args)
		// StartCPUProfileJS(this, args)
	}()
	return nil
}

func TraceProfile(d time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	err := trace.Start(&buf)
	if err != nil {
		return nil, err
	}
	time.Sleep(d)
	trace.Stop()
	return buf.Bytes(), nil
}

func TraceProfileJS(this js.Value, args []js.Value) interface{} {
	buf, err := TraceProfile(30 * time.Second)
	if err != nil {
		log.Error("Failed to create memory profile: ", err)
		return nil
	}
	StartDownload("", "hackpad-trace.pprof", buf)
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
	StartDownload("", "hackpad-mem.pprof", buf)
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
		StartDownload("", "hackpad-cpu.pprof", buf.Bytes())
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
