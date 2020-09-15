package plaineditor

import (
	"io/ioutil"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/ide"
	"github.com/johnstarich/go-wasm/log"
)

type textAreaBuilder struct {
}

func New() ide.EditorBuilder {
	return &textAreaBuilder{}
}

type textAreaEditor struct {
	elem     js.Value
	textarea js.Value
	filePath string
}

func (b *textAreaBuilder) New(elem js.Value) ide.Editor {
	elem.Set("innerHTML", `<textarea spellcheck=false></textarea>`)
	e := &textAreaEditor{
		elem:     elem,
		textarea: elem.Call("querySelector", "textarea"),
	}
	e.textarea.Call("addEventListener", "keydown", js.FuncOf(codeTyper))
	e.textarea.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go e.edited(func() string {
			return e.textarea.Get("value").String()
		})
		return nil
	}))
	return e
}

func (e *textAreaEditor) OpenFile(path string) error {
	e.filePath = path
	return e.ReloadFile()
}

func (e *textAreaEditor) ReloadFile() error {
	contents, err := ioutil.ReadFile(e.filePath)
	if err != nil {
		return err
	}
	e.textarea.Set("value", string(contents))
	return nil
}

func (e *textAreaEditor) edited(newContents func() string) {
	err := ioutil.WriteFile(e.filePath, []byte(newContents()), 0600)
	if err != nil {
		log.Errorf("Failed to write %s: %s", e.filePath, err.Error())
		return
	}
}

func (e *textAreaEditor) GetCursor() int {
	return e.textarea.Get("selectionStart").Int()
}

func (e *textAreaEditor) SetCursor(i int) error {
	e.textarea.Set("selectionStart", i)
	e.textarea.Set("selectionEnd", i)
	return nil
}
