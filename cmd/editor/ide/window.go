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
	NewPane() Editor
	NewConsole(rawName, name string, args ...string) (Console, error)
}

type window struct {
	elem js.Value
	consoleTabsElem,
	consolesElem,
	controlButtons,
	editorTabsElem,
	editorsElem,
	loadingElem js.Value

	currentConsoleTab, currentEditorTab int
	consoleTabButtons, editorTabButtons []js.Value
	consoleTabs, editorTabs             []js.Value
	consoles                            []Console
	editors                             []Editor

	editorBuilder  EditorBuilder
	consoleBuilder ConsoleBuilder

	showLoading atomic.Bool
}

func New(elem js.Value, editorBuilder EditorBuilder, consoleBuilder ConsoleBuilder, taskConsoleBuilder TaskConsoleBuilder) (Window, TaskConsole) {
	elem.Set("innerHTML", `
<div class="editors">
	<nav>
		<ul class="tab-buttons"></ul>
		<button class="tab-new"></button>
	</nav>
	<div class="tabs"></div>
</div>

<div class="controls">
	<button>build</button>
	<button>run</button>
	<button>fmt</button>
	<div class="loading-indicator"></div>
</div>

<div class="consoles">
	<nav>
		<ul class="tab-buttons"></ul>
		<button class="tab-new"></button>
	</nav>
	<div class="tabs"></div>
</div>

`)

	w := &window{
		elem:            elem,
		consoleBuilder:  consoleBuilder,
		consoleTabsElem: elem.Call("querySelector", ".consoles .tab-buttons"),
		consolesElem:    elem.Call("querySelector", ".consoles .tabs"),
		controlButtons:  elem.Call("querySelectorAll", ".controls button"),
		editorBuilder:   editorBuilder,
		editorTabsElem:  elem.Call("querySelector", ".editors .tab-buttons"),
		editorsElem:     elem.Call("querySelector", ".editors .tabs"),
		loadingElem:     elem.Call("querySelector", ".controls .loading-indicator"),
	}

	newTabElem := w.elem.Call("querySelector", ".editors .tab-new")
	newTabElem.Call("addEventListener", "click", js.FuncOf(func(js.Value, []js.Value) interface{} {
		filePickerTab := document.Call("createElement", "li")
		filePickerTab.Set("innerHTML", `<input type="text" placeholder="file_name.go" spellcheck=false />`)
		inputElem := filePickerTab.Call("querySelector", "input")

		removed := false
		var funcs []js.Func
		remove := func() {
			if removed {
				return
			}
			removed = true
			filePickerTab.Call("remove")
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
		addListener(filePickerTab, "keydown", func(args []js.Value) {
			if args[0].Get("key").String() != "Enter" {
				return
			}

			fileName := inputElem.Get("value").String()
			fileName = strings.TrimSpace(fileName)
			if fileName == "" {
				return
			}
			editor := w.NewPane()
			err := editor.OpenFile(fileName)
			if err != nil {
				log.Error(err)
			}
			remove()
		})
		addListener(inputElem, "blur", func([]js.Value) {
			remove()
		})
		w.editorTabsElem.Call("appendChild", filePickerTab)
		inputElem.Call("focus")
		return nil
	}))

	newTerminalElem := w.elem.Call("querySelector", ".consoles .tab-new")
	newTerminalElem.Call("addEventListener", "click", js.FuncOf(func(js.Value, []js.Value) interface{} {
		_, err := w.NewConsole("", "sh")
		if err != nil {
			log.Error(err)
		}
		return nil
	}))

	controlButtons := make(map[string]js.Value)
	for i := 0; i < w.controlButtons.Length(); i++ {
		button := w.controlButtons.Index(i)
		name := button.Get("textContent").String()
		controlButtons[name] = button
	}
	controlButtons["build"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.activateConsole(buildConsoleIndex)
		w.runGoProcess(w.currentConsole().(TaskConsole), "build", "-v", ".")
		return nil
	}))
	controlButtons["run"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.activateConsole(buildConsoleIndex)
		w.runPlayground(w.currentConsole().(TaskConsole))
		return nil
	}))
	controlButtons["fmt"].Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.activateConsole(buildConsoleIndex)
		w.runGoProcess(w.currentConsole().(TaskConsole), "fmt", ".").Then(func(_ js.Value) interface{} {
			err := w.currentEditor().ReloadFile()
			if err != nil {
				log.Error("Failed to reload file: ", err)
			}
			return nil
		})
		return nil
	}))

	taskConsole := w.newTaskConsole(taskConsoleBuilder)
	return w, taskConsole
}

func (w *window) NewPane() Editor {
	e := document.Call("createElement", "div")
	e.Set("className", "editor tab")
	w.editorsElem.Call("appendChild", e)
	w.editorTabs = append(w.editorTabs, e)
	editor := w.editorBuilder.New(e)
	w.editors = append(w.editors, editor)
	index := len(w.editors) - 1

	tabButton := document.Call("createElement", "li")
	tabButton.Set("innerHTML", `<button class="tab-button">New file</button>`)
	button := tabButton.Call("querySelector", "button")
	go watchTitles(editor.Titles(), button)
	button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.activateEditor(index)
		return nil
	}))
	w.editorTabsElem.Call("appendChild", tabButton)
	w.editorTabButtons = append(w.editorTabButtons, button)
	w.activateEditor(index)
	return editor
}

func watchTitles(titles <-chan string, elem js.Value) {
	for {
		title, ok := <-titles
		if !ok {
			return
		}
		elem.Set("innerText", title)
	}
}

func (w *window) NewConsole(rawName, name string, args ...string) (Console, error) {
	return w.newConsole(func(elem js.Value) (Console, error) {
		return w.consoleBuilder.New(elem, rawName, name, args...)
	})
}

func (w *window) newTaskConsole(builder TaskConsoleBuilder) TaskConsole {
	taskConsole, _ := w.newConsole(func(elem js.Value) (Console, error) {
		return builder.New(elem), nil
	})
	return taskConsole.(TaskConsole)
}

func (w *window) newConsole(makeConsole func(elem js.Value) (Console, error)) (Console, error) {
	contents := document.Call("createElement", "div")
	contents.Set("className", "console tab")
	w.consolesElem.Call("appendChild", contents)
	w.consoleTabs = append(w.consoleTabs, contents)
	console, err := makeConsole(contents)
	if err != nil {
		return nil, err
	}
	w.consoles = append(w.consoles, console)
	index := len(w.consoles) - 1

	tabButton := document.Call("createElement", "li")
	tabButton.Set("innerHTML", `<button class="tab-button">Terminal</button>`)
	button := tabButton.Call("querySelector", "button")
	go watchTitles(console.Titles(), button)
	button.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.activateConsole(index)
		return nil
	}))
	w.consoleTabsElem.Call("appendChild", tabButton)
	w.consoleTabButtons = append(w.consoleTabButtons, button)
	w.activateConsole(index)
	return console, nil
}
