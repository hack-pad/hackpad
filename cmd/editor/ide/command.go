package ide

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

const (
	goBinaryPath      = "/go/bin/js_wasm/go"
	buildConsoleIndex = 0
)

// runGoProcess optimizes runProcess by skipping the wait time for listing PATH directories on startup
func (w *window) runGoProcess(console TaskConsole, args ...string) promise.Promise {
	return w.runRawProcess(console, goBinaryPath, "go", args...)
}

func (w *window) runProcess(console TaskConsole, name string, args ...string) promise.Promise {
	return w.runRawProcess(console, name, name, args...)
}

func (w *window) runRawProcess(console TaskConsole, fullPath, name string, args ...string) promise.Promise {
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
	w.loadingElem.Get("classList").Call("add", "loading")
	defer func() {
		w.showLoading.Store(false)
		w.loadingElem.Get("classList").Call("remove", "loading")
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

func (w *window) activateEditor(tab int) {
	for ix, button := range w.editorTabButtons {
		if ix == tab {
			button.Get("classList").Call("add", "active")
		} else {
			button.Get("classList").Call("remove", "active")
		}
	}
	for ix, contents := range w.editorTabs {
		if ix == tab {
			contents.Get("classList").Call("add", "active")
		} else {
			contents.Get("classList").Call("remove", "active")
		}
	}
	firstInput := w.editorTabs[tab].Call("querySelector", "input, select, textarea")
	if firstInput.Truthy() {
		firstInput.Call("focus")
	}
	w.currentEditorTab = tab
}

func (w *window) currentEditor() Editor {
	return w.editors[w.currentEditorTab]
}

func (w *window) activateConsole(tab int) {
	for ix, button := range w.consoleTabButtons {
		if ix == tab {
			button.Get("classList").Call("add", "active")
		} else {
			button.Get("classList").Call("remove", "active")
		}
	}
	for ix, contents := range w.consoleTabs {
		if ix == tab {
			contents.Get("classList").Call("add", "active")
		} else {
			contents.Get("classList").Call("remove", "active")
		}
	}
	firstInput := w.consoleTabs[tab].Call("querySelector", "input, select, textarea")
	if firstInput.Truthy() {
		firstInput.Call("focus")
	}
	w.currentConsoleTab = tab
}

func (w *window) currentConsole() Console {
	return w.consoles[w.currentConsoleTab]
}

func (w *window) runPlayground(console TaskConsole) {
	w.runGoProcess(console, "build", "-v", ".").Then(func(_ js.Value) interface{} {
		return w.runProcess(console, "./playground")
	})
}
