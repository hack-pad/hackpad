//go:build js
// +build js

package main

import (
	"flag"
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
	"github.com/hack-pad/hackpad/cmd/editor/plaineditor"
	"github.com/hack-pad/hackpad/cmd/editor/taskconsole"
	"github.com/hack-pad/hackpad/cmd/editor/terminal"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
)

const (
	goBinaryPath = "/usr/local/go/bin/js_wasm/go"
)

func main() {
	editorID := flag.String("editor", "", "Editor element ID to attach")
	flag.Parse()

	if *editorID == "" {
		flag.Usage()
		os.Exit(2)
	}

	app := dom.GetDocument().GetElementByID(*editorID)
	app.AddClass("ide")
	globalEditorProps := js.Global().Get("editor")
	globalEditorProps.Set("profile", js.FuncOf(interop.ProfileJS))
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

	if _, err := tasks.Start(goBinaryPath, "go", "version"); err != nil {
		log.Error("Failed to start go version: ", err)
		return
	}

	if err := os.MkdirAll("playground", 0700); err != nil {
		log.Error("Failed to make playground dir", err)
		return
	}
	if err := os.Chdir("playground"); err != nil {
		log.Error("Failed to switch to playground dir", err)
		return
	}

	_, err := os.Stat("go.mod")
	makeNewModule := os.IsNotExist(err)
	if makeNewModule {
		_, err := tasks.Start(goBinaryPath, "go", "mod", "init", "playground")
		if err != nil {
			log.Error("Failed to start module init: ", err)
			return
		}
	}

	if _, err := os.Stat("main.go"); os.IsNotExist(err) {
		mainGoContents := `package main

import (
	"fmt"

	"github.com/johnstarich/go/datasize"
)

func main() {
	fmt.Println("Hello from Wasm!", datasize.Gigabytes(4))
}
`
		err := os.WriteFile("main.go", []byte(mainGoContents), 0600)
		if err != nil {
			log.Error("Failed to write to main.go: ", err)
			return
		}
	}

	if makeNewModule {
		_, err := tasks.Start(goBinaryPath, "go", "mod", "tidy")
		if err != nil {
			log.Error("Failed to start go mod tidy: ", err)
			return
		}
	}

	win.NewConsole()
	editor := win.NewEditor()
	err = editor.OpenFile("main.go")
	if err != nil {
		log.Error("Failed to open main.go in editor: ", err)
	}

	select {}
}
