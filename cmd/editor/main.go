package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall/js"

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
	app := document.Call("createElement", "div")
	app.Call("setAttribute", "id", "app")
	document.Get("body").Call("insertBefore", app, nil)

	app.Set("innerHTML", `
<h1>Go WASM Playground</h1>

<h3><pre>main.go</pre></h3>
<textarea></textarea>
<div class="controls">
	<button onclick='editor.run("go", "build", ".")'>build</button>
	<button onclick='editor.run("go", "run", ".")'>run</button>
	<button onclick='editor.run("go", "fmt", ".").then(() => editor.reload())'>fmt</button>
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

	js.Global().Set("editor", map[string]interface{}{
		"run": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			stringArgs := make([]string, len(args))
			for i := range args {
				stringArgs[i] = args[i].String()
			}
			var name string
			if len(stringArgs) > 0 {
				name = stringArgs[0]
				stringArgs = stringArgs[1:]
			}
			resolve, _, prom := promise.New()
			go func() {
				startProcess(name, stringArgs...)
				resolve(nil)
			}()
			return prom
		}),
		"reload": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			contents, err := ioutil.ReadFile("main.go")
			if err != nil {
				log.Error(err)
				return nil
			}
			editorElem.Set("value", string(contents))
			return nil
		}),
	})
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
	cmd := exec.Command("go", "mod", "init", "playground")
	err := cmd.Start()
	if err != nil {
		log.Error("Failed to run go mod init", err)
		return
	}

	mainGoContents := `package main

func main() {
	println("Hello from WASM!")
}
`
	editorElem.Set("value", mainGoContents)
	go edited(func() string { return mainGoContents })
	select {}
}

func startProcess(name string, args ...string) {
	if !showLoading.CAS(false, true) {
		return
	}
	loadingElem.Get("classList").Call("add", "loading")
	defer func() {
		showLoading.Store(false)
		loadingElem.Get("classList").Call("remove", "loading")
	}()

	stdout := newElementWriter(consoleElem, "")
	stderr := newElementWriter(consoleElem, "stderr")

	_, _ = stdout.WriteString(fmt.Sprintf("$ %s %s\n", name, strings.Join(args, " ")))

	cmd := exec.Command(name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		_, _ = stderr.WriteString(err.Error() + "\n")
	}
}

func edited(newContents func() string) {
	err := ioutil.WriteFile("main.go", []byte(newContents()), 0700)
	if err != nil {
		log.Error("Failed to write main.go: ", err.Error())
		return
	}
}
