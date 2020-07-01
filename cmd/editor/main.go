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
	<button onclick='editor.run("go", "build", "-v", ".")'>build</button>
	<button onclick='editor.run("go", "build", "-v", ".")
						.then(() => editor.run("./playground"))
						.catch(() => {})'>run</button>
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
			resolve, reject, prom := promise.New()
			go func() {
				success := startProcess(name, stringArgs...)
				if success {
					resolve(nil)
				} else {
					reject(nil)
				}
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
	editorElem.Call("addEventListener", "keydown", jsCodeTyper())
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		log.Error("Failed to run go mod init", err)
		return
	}

	mainGoContents := `package main

import (
	"fmt"

	_ "github.com/johnstarich/go/datasize"
)

func main() {
	fmt.Println("Hello from WASM!")
}
`
	editorElem.Set("value", mainGoContents)
	go edited(func() string { return mainGoContents })
	select {}
}

func startProcess(name string, args ...string) (success bool) {
	if !showLoading.CAS(false, true) {
		return false
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

	err := cmd.Start()
	if err != nil {
		_, _ = stderr.WriteString("Failed to start process: " + err.Error() + "\n")
		return false
	}
	err = cmd.Wait()
	if err != nil {
		_, _ = stderr.WriteString(err.Error() + "\n")
	}
	return err == nil
}

func edited(newContents func() string) {
	err := ioutil.WriteFile("main.go", []byte(newContents()), 0600)
	if err != nil {
		log.Error("Failed to write main.go: ", err.Error())
		return
	}
}

func jsCodeTyper() js.Value {
	// add raw JS func to handle typing events, to avoid slow WASM wake-ups
	return js.Global().Call("Function", `
"use strict"
const e = arguments[0]

if (e.code === 'Tab') {
	e.preventDefault()
	document.execCommand("insertText", false, "\t")
	return
}

const val = e.target.value
const sel = e.target.selectionStart

function parseBracket(s) {
	switch (s) {
	case "{":
		return {opener: true, closer: false, next: "}"}
	case "}":
		return {opener: false, closer: true, next: ""}
	case "[":
		return {opener: true, closer: false, next: "]"}
	case "]":
		return {opener: false, closer: true, next: ""}
	case "(":
		return {opener: true, closer: false, next: ")"}
	case ")":
		return {opener: false, closer: true, next: ""}
	case '"':
		return {opener: true, closer: true, next: '"'}
	case "'":
		return {opener: true, closer: true, next: "'"}
	default:
		return {next: ""}
	}
}

if (e.code === 'Enter') {
	if (e.metaKey) {
		e.preventDefault()
		editor.run("go", "run", ".")
		return
	}

	const lastLine = val.slice(0, sel).lastIndexOf("\n")
	if (lastLine !== -1) {
		const leadingChars = val.slice(lastLine+1, sel)
		const leadingSpace = leadingChars.slice(0, leadingChars.length - leadingChars.trimStart().length)
		const prevChar = leadingChars.slice(-1)
		const nextChar = val.slice(sel, sel+1)
		let newLinePrefix = "\n"+leadingSpace
		let newLineSuffix = ""
		const prevBracket = parseBracket(prevChar)
		const nextBracket = parseBracket(nextChar)
		console.log("opener", prevChar, prevBracket, nextBracket)
		if (prevBracket.opener) {
			newLinePrefix += "\t"
			if (nextBracket.closer) {
				newLineSuffix += "\n"+leadingSpace
			}
		}
		document.execCommand("insertText", false, newLinePrefix+newLineSuffix)
		e.target.selectionStart = sel + newLinePrefix.length
		e.target.selectionEnd = sel + newLinePrefix.length
		e.preventDefault()
	}
	return
}

if (e.code === 'Backspace') {
	const prevChar = val.slice(sel-1, sel)
	const nextChar = val.slice(sel, sel+1)
	if (parseBracket(prevChar).next === nextChar) {
		document.execCommand("forwardDelete", false)
	}
	return
}

if (sel !== e.target.selectionEnd) {
	return
}

const closer = parseBracket(e.key).next
const afterSel = val.slice(sel).slice(0, 1)
if (closer !== "" && afterSel !== closer) {
	e.preventDefault()
	document.execCommand("insertText", false, e.key+closer)
	e.target.selectionStart = sel+1
	e.target.selectionEnd = sel+1
	return
}

const nextChar = val.slice(sel, sel+1)
if (e.key === nextChar) {
	if (parseBracket(nextChar).closer) {
		e.preventDefault()
		e.target.selectionStart = sel+1
		e.target.selectionEnd = sel+1
		return
	}
}
`)
}
