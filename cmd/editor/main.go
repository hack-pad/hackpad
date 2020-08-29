package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
	"go.uber.org/atomic"
)

var (
	showLoading = atomic.NewBool(false)
	loadingElem js.Value
	consoleElem js.Value

	document = js.Global().Get("document")
)

func main() {
	js.Global().Set("editor", map[string]interface{}{
		"profile": js.FuncOf(interop.MemoryProfile),
	})

	app := document.Call("createElement", "div")
	app.Call("setAttribute", "id", "app")
	document.Get("body").Call("insertBefore", app, nil)

	app.Set("innerHTML", `
<h1>Go WASM Playground</h1>

<h3><pre>main.go</pre></h3>
<textarea></textarea>
<div class="controls">
	<button>build</button>
	<button>run</button>
	<button>fmt</button>
	<div class="loading-indicator"></div>
</div>
<div class="console">
	<h3>Console</h3>
	<pre class="console-output"></pre>
</div>
`)
	loadingElem = app.Call("querySelector", ".controls .loading-indicator")
	consoleElem = app.Call("querySelector", ".console-output")
	editorElem := app.Call("querySelector", "textarea")
	controlButtonElems := app.Call("querySelectorAll", ".controls button")

	controlButtons := make(map[string]js.Value)
	for i := 0; i < controlButtonElems.Length(); i++ {
		button := controlButtonElems.Index(i)
		name := button.Get("textContent").String()
		controlButtons[name] = button
	}
	controlButtons["build"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		runGoProcess("build", "-v", ".")
		return nil
	}))
	controlButtons["run"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		runPlayground()
		return nil
	}))
	controlButtons["fmt"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		runGoProcess("fmt", ".").Then(func(_ js.Value) interface{} {
			contents, err := ioutil.ReadFile("main.go")
			if err != nil {
				log.Error(err)
				return nil
			}
			editorElem.Set("value", string(contents))
			return nil
		})
		return nil
	}))

	editorElem.Call("addEventListener", "keydown", js.FuncOf(codeTyper))
	editorElem.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go edited(func() string {
			return editorElem.Get("value").String()
		})
		return nil
	}))

	if err := os.Mkdir("playground", 0700); err != nil {
		log.Error("Failed to make playground dir", err)
		return
	}
	if err := os.Chdir("playground"); err != nil {
		log.Error("Failed to switch to playground dir", err)
		return
	}
	runGoProcess("mod", "init", "playground").Then(func(value js.Value) interface{} {
		return runGoProcess("version")
	})

	mainGoContents := `package main

import (
	"fmt"

	"github.com/johnstarich/go/datasize"
)

func main() {
	fmt.Println("Hello from WASM!", datasize.Gigabytes(4))
}
`
	editorElem.Set("value", mainGoContents)
	go edited(func() string { return mainGoContents })
	select {}
}

// runGoProcess optimizes runProcess by skipping the wait time for listing PATH directories on startup
func runGoProcess(args ...string) promise.Promise {
	return runRawProcess("/go/bin/js_wasm/go", "go", args...)
}

func runProcess(name string, args ...string) promise.Promise {
	return runRawProcess(name, name, args...)
}

func runRawProcess(fullPath, name string, args ...string) promise.Promise {
	resolve, reject, prom := promise.New()
	go func() {
		var success bool
		var elapsed time.Duration
		defer func() {
			log.Printf("Process [%s %s] finished: %6.2fs", name, strings.Join(args, " "), elapsed.Seconds())
		}()
		success, elapsed = startProcess(fullPath, name, args...)
		if success {
			resolve(nil)
		} else {
			reject(nil)
		}
	}()
	return prom
}

func startProcess(rawPath, name string, args ...string) (success bool, elapsed time.Duration) {
	if !showLoading.CAS(false, true) {
		return false, 0
	}
	startTime := time.Now()
	loadingElem.Get("classList").Call("add", "loading")
	defer func() {
		showLoading.Store(false)
		loadingElem.Get("classList").Call("remove", "loading")
	}()

	stdout := newElementWriter(consoleElem, "")
	stderr := newElementWriter(consoleElem, "stderr")
	note := newElementWriter(consoleElem, "note")

	_, _ = stdout.WriteString(fmt.Sprintf("$ %s %s\n", name, strings.Join(args, " ")))

	cmd := exec.Command(rawPath, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Start()
	if err != nil {
		_, _ = stderr.WriteString("Failed to start process: " + err.Error() + "\n")
		return false, 0
	}
	err = cmd.Wait()
	if err != nil {
		_, _ = stderr.WriteString(err.Error() + "\n")
	}
	elapsed = time.Since(startTime)
	_, _ = note.WriteString(fmt.Sprintf("(%.2fs)\n", elapsed.Seconds()))
	return err == nil, elapsed
}

func edited(newContents func() string) {
	err := ioutil.WriteFile("main.go", []byte(newContents()), 0600)
	if err != nil {
		log.Error("Failed to write main.go: ", err.Error())
		return
	}
}

func runPlayground() {
	runGoProcess("build", "-v", ".").Then(func(_ js.Value) interface{} {
		return runProcess("./playground")
	})
}
