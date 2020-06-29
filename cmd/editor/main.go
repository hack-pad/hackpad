package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall/js"

	"go.uber.org/atomic"
)

var (
	showLoading = atomic.NewBool(false)
	loadingElem js.Value
)

func printerr(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
}

func main() {
	document := js.Global().Get("document")
	app := document.Call("createElement", "div")
	app.Call("setAttribute", "id", "app")
	document.Get("body").Call("insertBefore", app, nil)

	app.Set("innerHTML", `
<h1>Go WASM Playground</h1>

<textarea> </textarea>
<div class="controls">
	<button onclick='editor.build()'>Build</button>
	<button onclick='editor.run()'>Run</button>
	<div class="loading-indicator" />
</div>
`)
	loadingElem = app.Call("querySelector", "#app .controls .loading-indicator")

	js.Global().Set("editor", map[string]interface{}{
		"build": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go build()
			return nil
		}),
		"run": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			go run()
			return nil
		}),
	})
	editorElem := app.Call("querySelector", "textarea")
	editorElem.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go edited(editorElem.Get("value").String())
		return nil
	}))

	if err := os.Mkdir("playground", 0700); err != nil {
		printerr("Failed to make playground dir", err)
		return
	}
	if err := os.Chdir("playground"); err != nil {
		printerr("Failed to switch to playground dir", err)
		return
	}
	cmd := exec.Command("go", "mod", "init", "playground")
	err := cmd.Start()
	if err != nil {
		printerr("Failed to run go mod init", err)
		return
	}

	mainGoContents := `package main

func main() {
	println("Hello from WASM!")
}
`
	editorElem.Set("value", mainGoContents)
	go edited(mainGoContents)
	select {}
}

func startProcess() (shouldRun bool) {
	shouldRun = showLoading.CAS(false, true)
	if !shouldRun {
		return
	}

	loadingElem.Get("classList").Call("add", "loading")
	return
}

func endProcess() {
	showLoading.Store(false)
	loadingElem.Get("classList").Call("remove", "loading")
}

func build() {
	if !startProcess() {
		return
	}
	defer endProcess()

	cmd := exec.Command("go", "build", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printerr("Failed to build:", err)
	}
}

func run() {
	if !startProcess() {
		return
	}
	defer endProcess()

	cmd := exec.Command("go", "run", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printerr("Failed to run:", err)
	}
}

func edited(newContents string) {
	err := ioutil.WriteFile("main.go", []byte(newContents), 0700)
	if err != nil {
		printerr("Failed to write main.go: ", err.Error())
		return
	}
}
