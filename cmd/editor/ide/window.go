package ide

import (
	"go/format"
	"io/ioutil"
	"runtime/debug"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/log"
	"go.uber.org/atomic"
)

var (
	document = js.Global().Get("document")
)

type Window interface {
	NewEditor() Editor
	NewConsole() Console
}

type window struct {
	elem js.Value
	panesElem,
	controlButtons,
	loadingElem js.Value

	consoleBuilder ConsoleBuilder
	consoles       []Console
	consolesPane   *TabPane
	editorBuilder  EditorBuilder
	editors        []Editor
	editorsPane    *TabPane

	showLoading atomic.Bool
}

func New(elem js.Value, editorBuilder EditorBuilder, consoleBuilder ConsoleBuilder, taskConsoleBuilder TaskConsoleBuilder) (Window, TaskConsole) {
	elem.Set("innerHTML", `
<header>
	<h1 class="app-title">
		<span class="app-title-go">go</span>
		<span class="app-title-wasm">wasm</span>
	</h1>
	<div class="controls">
		<button class="control" title="build"><span class="fa fa-hammer"></span></button>
		<button class="control" title="run"><span class="fa fa-play"></span></button>
		<button class="control" title="gofmt"><span class="fa fa-magic"></span></button>
		<div class="loading-indicator"></div>
		<div class="spacer"></div>
		<div class="control">
			<a target="_blank" title="Go WASM on GitHub" href="https://github.com/johnstarich/go-wasm">
				<span class="fab fa-github"></span>
			</a>
		</div>
	</div>
</header>

<div class="panes">
</div>
`)

	w := &window{
		consoleBuilder: consoleBuilder,
		controlButtons: elem.Call("querySelectorAll", ".controls button"),
		editorBuilder:  editorBuilder,
		elem:           elem,
		loadingElem:    elem.Call("querySelector", ".controls .loading-indicator"),
		panesElem:      elem.Call("querySelector", ".panes"),
	}

	w.editorsPane = NewTabPane(TabOptions{NoFocus: true}, func(id int, title, contents js.Value) Tabber {
		contents.Get("classList").Call("add", "editor")
		editor := w.editorBuilder.New(contents)
		w.editors = append(w.editors, editor)

		title.Set("innerHTML", `<input type="text" class="editor-file-picker" placeholder="file_name.go" spellcheck=false />`)
		inputElem := title.Call("querySelector", "input")
		inputElem.Call("focus")

		var keydownFn js.Func
		keydownFn = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			defer func() {
				if r := recover(); r != nil {
					log.Print("recovered from panic:", r, string(debug.Stack()))
				}
			}()
			event := args[0]
			if event.Get("key").String() != "Enter" {
				return nil
			}
			event.Call("preventDefault")
			event.Call("stopPropagation")

			fileName := inputElem.Get("value").String()
			fileName = strings.TrimSpace(fileName)
			if fileName == "" {
				return nil
			}
			title.Set("innerText", "New file")
			err := editor.OpenFile(fileName)
			if err != nil {
				log.Error(err)
			}
			w.editorsPane.focusID(id)
			keydownFn.Release()
			return nil
		})
		title.Call("addEventListener", "keydown", keydownFn)
		inputElem.Call("addEventListener", "blur", interop.SingleUseFunc(func(js.Value, []js.Value) interface{} {
			titleText := title.Get("innerText")
			if titleText.Truthy() && titleText.String() != "New file" {
				w.editorsPane.closeTabID(id)
			}
			return nil
		}))
		return editor
	}, func(closedIndex int) {
		var newEditors []Editor
		newEditors = append(newEditors, w.editors[:closedIndex]...)
		newEditors = append(newEditors, w.editors[closedIndex+1:]...)
		w.editors = newEditors
	})
	w.panesElem.Call("appendChild", w.editorsPane)

	w.consolesPane = NewTabPane(TabOptions{}, func(_ int, _, contents js.Value) Tabber {
		console, err := w.consoleBuilder.New(contents, "", "sh")
		if err != nil {
			log.Error(err)
		}
		w.consoles = append(w.consoles, console)
		return console
	}, func(closedIndex int) {
		var newConsoles []Console
		newConsoles = append(newConsoles, w.consoles[:closedIndex]...)
		newConsoles = append(newConsoles, w.consoles[closedIndex+1:]...)
		w.consoles = newConsoles
	})
	w.panesElem.Call("appendChild", w.consolesPane)

	w.controlButtons.Index(0).Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runGoProcess(console.(TaskConsole), "build", "-v", ".")
		return nil
	}))
	w.controlButtons.Index(1).Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runPlayground(console.(TaskConsole))
		return nil
	}))
	w.controlButtons.Index(2).Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		ix := w.editorsPane.currentTab
		if ix < 0 || ix >= len(w.editorsPane.tabs) {
			return nil
		}

		editor := w.editors[ix]
		path := editor.CurrentFile()
		if path == "" {
			return nil
		}

		src, err := ioutil.ReadFile(path)
		if err != nil {
			log.Errorf("Failed to read Go file %q: %v", path, err)
			return nil
		}
		out, err := format.Source(src)
		if err != nil {
			log.Errorf("Failed to format Go file %q: %v", path, err)
			return nil
		}
		err = ioutil.WriteFile(path, out, 0)
		if err != nil {
			log.Errorf("Failed to write Go file %q: %v", path, err)
			return nil
		}
		err = editor.ReloadFile()
		if err != nil {
			log.Errorf("Failed to reload Go file %q: %v", path, err)
			return nil
		}
		return nil
	}))

	taskConsole := w.consolesPane.NewTab(TabOptions{NoClose: true}, func(_ int, _, contents js.Value) Tabber {
		c := taskConsoleBuilder.New(contents)
		w.consoles = append(w.consoles, c)
		return c
	}).(TaskConsole)
	return w, taskConsole
}

func (w *window) NewEditor() Editor {
	return w.editorsPane.NewDefaultTab(TabOptions{}).(Editor)
}

func (w *window) NewConsole() Console {
	return w.consolesPane.NewDefaultTab(TabOptions{}).(Console)
}
