package main

import (
	"flag"
	"io/ioutil"
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/ide"
	"github.com/johnstarich/go-wasm/cmd/editor/plaineditor"
	"github.com/johnstarich/go-wasm/cmd/editor/taskconsole"
	"github.com/johnstarich/go-wasm/cmd/editor/terminal"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
)

var (
	document = js.Global().Get("document")
)

const (
	goBinaryPath = "/go/bin/js_wasm/go"
)

func main() {
	editorID := flag.String("editor", "", "Editor element ID to attach")
	flag.Parse()

	if *editorID == "" {
		flag.Usage()
		os.Exit(2)
	}

	app := document.Call("querySelector", "#"+*editorID)
	app.Set("className", "ide")
	globalEditorProps := js.Global().Get("editor")
	globalEditorProps.Set("profile", js.FuncOf(interop.MemoryProfile))
	newEditor := globalEditorProps.Get("newEditor")
	var editorBuilder ide.EditorBuilder = editorJSFunc(newEditor)
	if !newEditor.Truthy() {
		editorBuilder = plaineditor.New()
	}
	newXTermFunc := globalEditorProps.Get("newTerminal")
	if !newXTermFunc.Truthy() {
		panic("window.editor.newTerminal must be set")
	}

	consoleBuilder := terminal.New(newXTermFunc)
	taskConsoleBuilder := taskconsole.New()
	win, tasks := ide.New(app, editorBuilder, consoleBuilder, taskConsoleBuilder)

	if err := os.MkdirAll("playground", 0700); err != nil {
		log.Error("Failed to make playground dir", err)
		return
	}
	if err := os.Chdir("playground"); err != nil {
		log.Error("Failed to switch to playground dir", err)
		return
	}

	if _, err := tasks.Start(goBinaryPath, "go", "mod", "init", "playground"); err != nil {
		log.Error("Failed to start module init: ", err)
		return
	}

	if _, err := tasks.Start(goBinaryPath, "go", "version"); err != nil {
		log.Error("Failed to start go version: ", err)
		return
	}

	mainGoContents := `package main

import (
	"fmt"

	"github.com/johnstarich/go/datasize"
)

func main() {
	fmt.Println("Hello from Wasm!", datasize.Gigabytes(4))
}
`
	err := ioutil.WriteFile("main.go", []byte(mainGoContents), 0600)
	if err != nil {
		log.Error("Failed to write to main.go: ", err)
		return
	}

	win.NewConsole()
	editor := win.NewEditor()
	editor.OpenFile("main.go")

	select {}
}
