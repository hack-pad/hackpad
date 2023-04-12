//go:build js
// +build js

package ide

import (
	"strings"
	"time"

	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/promise"
)

const (
	goBinaryPath      = "/usr/local/go/bin/js_wasm/go"
	buildConsoleIndex = 0
)

// runGoProcess optimizes runProcess by skipping the wait time for listing PATH directories on startup
func (w *window) runGoProcess(console TaskConsole, args ...string) promise.JS {
	return w.runRawProcess(console, goBinaryPath, "go", args...)
}

func (w *window) runProcess(console TaskConsole, name string, args ...string) promise.JS {
	return w.runRawProcess(console, name, name, args...)
}

func (w *window) runRawProcess(console TaskConsole, fullPath, name string, args ...string) promise.JS {
	resolve, reject, prom := promise.New()
	go func() {
		var success bool
		var elapsed time.Duration
		defer func() {
			log.Printf("Process [%s %s] finished: %6.2fs", name, strings.Join(args, " "), elapsed.Seconds())
		}()
		success, elapsed = w.startProcess(console, fullPath, name, args...)
		if success {
			resolve(nil)
		} else {
			reject(nil)
		}
	}()
	return prom
}

func (w *window) startProcess(console TaskConsole, rawPath, name string, args ...string) (success bool, elapsed time.Duration) {
	if !w.showLoading.CAS(false, true) {
		return false, 0
	}
	startTime := time.Now()
	w.loadingElem.AddClass("loading")
	defer func() {
		w.showLoading.Store(false)
		w.loadingElem.RemoveClass("loading")
	}()

	ctx, err := console.Start(rawPath, name, args...)
	if err != nil {
		log.Error("Failed to start process: " + err.Error() + "\n")
		return false, 0
	}
	<-ctx.Done()
	commandErr := ctx.Err()
	if commandErr != nil {
		log.Error(commandErr.Error())
	}
	elapsed = time.Since(startTime)
	return commandErr == nil, elapsed
}

func (w *window) runPlayground(console TaskConsole) {
	w.runGoProcess(console, "build", "-v", ".").Then(func(_ interface{}) interface{} {
		return w.runProcess(console, "./playground").JSValue()
	})
}
