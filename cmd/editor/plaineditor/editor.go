//go:build js
// +build js

package plaineditor

import (
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/cmd/editor/ide"
	"github.com/hack-pad/hackpad/internal/log"
)

type textAreaBuilder struct {
}

func New() ide.EditorBuilder {
	return &textAreaBuilder{}
}

type textAreaEditor struct {
	elem      *dom.Element
	textarea  *dom.Element
	filePath  string
	titleChan chan string
}

func (b *textAreaBuilder) New(elem *dom.Element) ide.Editor {
	elem.SetInnerHTML(`<textarea spellcheck=false></textarea>`)
	e := &textAreaEditor{
		elem:      elem,
		textarea:  elem.QuerySelector("textarea"),
		titleChan: make(chan string, 1),
	}
	e.textarea.AddEventListener("keydown", codeTyper)
	e.textarea.AddEventListener("input", func(event js.Value) {
		go e.edited(e.textarea.Value)
	})
	return e
}

func (e *textAreaEditor) OpenFile(path string) error {
	e.filePath = path
	e.titleChan <- path
	return e.ReloadFile()
}

func (e *textAreaEditor) CurrentFile() string {
	return e.filePath
}

func (e *textAreaEditor) ReloadFile() error {
	contents, err := os.ReadFile(e.filePath)
	if err != nil {
		return err
	}
	e.textarea.SetValue(string(contents))
	return nil
}

func (e *textAreaEditor) edited(newContents func() string) {
	err := os.WriteFile(e.filePath, []byte(newContents()), 0600)
	if err != nil {
		log.Errorf("Failed to write %s: %s", e.filePath, err.Error())
		return
	}
}

func (e *textAreaEditor) GetCursor() int {
	return e.textarea.GetProperty("selectionStart").Int()
}

func (e *textAreaEditor) SetCursor(i int) error {
	v := js.ValueOf(i)
	e.textarea.SetProperty("selectionStart", v)
	e.textarea.SetProperty("selectionEnd", v)
	return nil
}

func (e *textAreaEditor) Titles() <-chan string {
	return e.titleChan
}
