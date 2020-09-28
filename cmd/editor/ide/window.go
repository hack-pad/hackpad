package ide

import (
	"strings"
	"syscall/js"

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
<div class="controls">
	<button>build</button>
	<button>run</button>
	<button>fmt</button>
	<div class="loading-indicator"></div>
</div>

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

	w.editorsPane = NewTabPane(TabOptions{NoFocus: true}, func(title, contents js.Value) Tabber {
		contents.Get("classList").Call("add", "editor")
		editor := w.editorBuilder.New(contents)
		index := len(w.editors)
		w.editors = append(w.editors, editor)

		title.Set("innerHTML", `<input type="text" placeholder="file_name.go" spellcheck=false />`)
		inputElem := title.Call("querySelector", "input")
		inputElem.Call("focus")

		removed := false
		var funcs []js.Func
		remove := func() {
			if removed {
				return
			}
			removed = true
			for _, f := range funcs {
				f.Release()
			}
		}
		addListener := func(elem js.Value, event string, fn func([]js.Value)) {
			f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				fn(args)
				return nil
			})
			funcs = append(funcs, f)
			elem.Call("addEventListener", event, f)
		}
		addListener(title, "keydown", func(args []js.Value) {
			event := args[0]
			if event.Get("key").String() != "Enter" {
				return
			}
			event.Call("preventDefault")
			event.Call("stopPropagation")

			fileName := inputElem.Get("value").String()
			fileName = strings.TrimSpace(fileName)
			if fileName == "" {
				return
			}
			title.Set("innerText", "New file")
			err := editor.OpenFile(fileName)
			if err != nil {
				log.Error(err)
			}
			w.editorsPane.Focus(index)
		})
		addListener(inputElem, "blur", func([]js.Value) {
			remove()
		})
		return editor
	}, func(closedIndex int) {
		var newEditors []Editor
		newEditors = append(newEditors, w.editors[:closedIndex]...)
		newEditors = append(newEditors, w.editors[closedIndex+1:]...)
		w.editors = newEditors
	})
	w.panesElem.Call("appendChild", w.editorsPane)

	w.consolesPane = NewTabPane(TabOptions{}, func(_, contents js.Value) Tabber {
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

	controlButtons := make(map[string]js.Value)
	for i := 0; i < w.controlButtons.Length(); i++ {
		button := w.controlButtons.Index(i)
		name := button.Get("textContent").String()
		controlButtons[name] = button
	}
	controlButtons["build"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runGoProcess(console.(TaskConsole), "build", "-v", ".")
		return nil
	}))
	controlButtons["run"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runPlayground(console.(TaskConsole))
		return nil
	}))
	controlButtons["fmt"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runGoProcess(console.(TaskConsole), "fmt", ".").Then(func(_ js.Value) interface{} {
			for _, editor := range w.editors {
				err := editor.ReloadFile()
				if err != nil {
					log.Error("Failed to reload file: ", err)
				}
			}
			return nil
		})
		return nil
	}))

	taskConsole := w.consolesPane.NewTab(TabOptions{}, func(_, contents js.Value) Tabber {
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
