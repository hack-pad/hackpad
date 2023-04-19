//go:build js
// +build js

package ide

import (
	_ "embed"
	"go/format"
	"os"
	"strings"
	"syscall/js"

	"github.com/avct/uasurfer"
	"github.com/hack-pad/hackpad/cmd/editor/css"
	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/internal/log"
	"go.uber.org/atomic"
)

var (
	//go:embed window.html
	windowHTML string
	//go:embed window.css
	windowCSS string
)

type Window interface {
	NewEditor() Editor
	NewConsole() Console
}

type window struct {
	*dom.Element

	panesElem,
	loadingElem *dom.Element
	controlButtons []*dom.Element

	consoleBuilder ConsoleBuilder
	consoles       []Console
	consolesPane   *TabPane
	editorBuilder  EditorBuilder
	editors        []Editor
	editorsPane    *TabPane

	showLoading atomic.Bool
}

func New(elem *dom.Element, editorBuilder EditorBuilder, consoleBuilder ConsoleBuilder, taskConsoleBuilder TaskConsoleBuilder) (Window, TaskConsole) {
	css.Add(windowCSS)
	elem.SetInnerHTML(windowHTML)

	w := &window{
		Element:        elem,
		consoleBuilder: consoleBuilder,
		controlButtons: elem.QuerySelectorAll(".controls button"),
		editorBuilder:  editorBuilder,
		loadingElem:    elem.QuerySelector(".controls .loading-indicator"),
		panesElem:      elem.QuerySelector(".panes"),
	}

	w.editorsPane = NewTabPane(TabOptions{NoFocus: true}, w.makeDefaultEditor, w.closedEditor)
	w.panesElem.AppendChild(w.editorsPane.Element)

	w.consolesPane = NewTabPane(TabOptions{}, func(_ int, _, contents *dom.Element) Tabber {
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
	w.panesElem.AppendChild(w.consolesPane.Element)

	w.controlButtons[0].AddEventListener("click", func(event js.Value) {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runGoProcess(console.(TaskConsole), "build", "-v", ".")
	})
	w.controlButtons[1].AddEventListener("click", func(event js.Value) {
		w.consolesPane.Focus(buildConsoleIndex)
		console := w.consoles[buildConsoleIndex]
		w.runPlayground(console.(TaskConsole))
	})
	w.controlButtons[2].AddEventListener("click", func(event js.Value) {
		ix := w.editorsPane.currentTab
		if ix < 0 || ix >= len(w.editorsPane.tabs) {
			return
		}

		editor := w.editors[ix]
		path := editor.CurrentFile()
		if path == "" {
			return
		}

		go func() {
			src, err := os.ReadFile(path)
			if err != nil {
				log.Errorf("Failed to read Go file %q: %v", path, err)
				return
			}
			out, err := format.Source(src)
			if err != nil {
				log.Errorf("Failed to format Go file %q: %v", path, err)
				return
			}
			err = os.WriteFile(path, out, 0)
			if err != nil {
				log.Errorf("Failed to write Go file %q: %v", path, err)
				return
			}
			err = editor.ReloadFile()
			if err != nil {
				log.Errorf("Failed to reload Go file %q: %v", path, err)
				return
			}
		}()
	})

	controls := elem.QuerySelector(".controls")
	settings := newSettings()
	controls.AppendChild(settings)

	if !isCompatibleBrowser() {
		dialogElem := dom.New("div")
		dialogElem.AddClass("compatibility-warning-dialog")
		dialogElem.SetInnerHTML(`
			<p>Hackpad may not work reliably in your browser.</p>
			<p>If you're experience any issues, try a recent version of Chrome or Firefox on a device with enough memory, like a PC.</p>
		`)

		warningElem := dom.New("button")
		warningElem.AddClass("control")
		warningElem.AddClass("compatibility-warning")
		warningElem.SetAttribute("title", "Show browser compatibility warning")
		warningElem.SetInnerHTML(`<span class="fa fa-exclamation-triangle"></span>`)
		warningElem.AddEventListener("click", func(event js.Value) {
			dialogElem.ToggleClass("compatibility-warning-show")
		})

		dom.Body().InsertBefore(dialogElem, dom.Body().FirstChild())
		controls.AppendChild(warningElem)
	}

	taskConsole := w.consolesPane.NewTab(TabOptions{NoClose: true}, func(_ int, _, contents *dom.Element) Tabber {
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

func isCompatibleBrowser() bool {
	userAgentStr := js.Global().Get("navigator").Get("userAgent").String()
	userAgent := uasurfer.Parse(userAgentStr)
	if userAgent.DeviceType != uasurfer.DeviceComputer {
		return false
	}
	switch userAgent.Browser.Name {
	case uasurfer.BrowserChrome, uasurfer.BrowserFirefox:
		return true
	}
	return false
}

func (w *window) makeDefaultEditor(id int, title, contents *dom.Element) Tabber {
	contents.AddClass("editor")
	editor := w.editorBuilder.New(contents)
	w.editors = append(w.editors, editor)

	title.SetInnerHTML(`<input type="text" class="editor-file-picker" placeholder="file_name.go" spellcheck=false />`)
	inputElem := title.QuerySelector("input")
	inputElem.Focus()

	blurListener := inputElem.AddEventListener("blur", func(js.Value) {
		w.editorsPane.closeTabID(id)
	})
	title.AddEventListener("keydown", func(event js.Value) {
		if event.Get("key").String() != "Enter" {
			return
		}
		event.Call("preventDefault")
		event.Call("stopPropagation")

		fileName := inputElem.Value()
		fileName = strings.TrimSpace(fileName)
		if fileName == "" {
			return
		}
		inputElem.RemoveEventListener("blur", blurListener)
		title.SetInnerText("New file") // setting inner text triggers onblur because the input HTML is about to be removed
		go func() {
			err := editor.OpenFile(fileName)
			if err != nil {
				log.Error(err)
			}
			w.editorsPane.focusID(id)
		}()
	})
	return editor
}

func (w *window) closedEditor(closedIndex int) {
	var newEditors []Editor
	newEditors = append(newEditors, w.editors[:closedIndex]...)
	newEditors = append(newEditors, w.editors[closedIndex+1:]...)
	w.editors = newEditors
}
