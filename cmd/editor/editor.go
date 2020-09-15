package main

import (
	"io/ioutil"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/ide"
	"github.com/johnstarich/go-wasm/cmd/editor/plaineditor"
	"github.com/johnstarich/go-wasm/log"
)

var (
	editorBuilder ide.EditorBuilder = plaineditor.New()
)

type editorJSFunc js.Value

func (e editorJSFunc) New(elem js.Value) ide.Editor {
	editorElem := js.Value(e).Invoke(elem)
	return &jsEditor{
		elem:      editorElem,
		titleChan: make(chan string, 1),
	}
}

func setEditorFunc(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		log.Error("Invalid number of args. Expected function to attach to a DOM element.")
		return nil
	}

	editorBuilder = editorJSFunc(args[0])
	// TODO re-init existing editors?
	return nil
}

type jsEditor struct {
	elem      js.Value
	filePath  string
	titleChan chan string
}

func (j *jsEditor) onEdit(js.Value, []js.Value) interface{} {
	contents := j.elem.Call("getContents").String()
	err := ioutil.WriteFile(j.filePath, []byte(contents), 0700)
	if err != nil {
		log.Error("Failed to write file contents: ", err)
	}
	return nil
}

func (j *jsEditor) OpenFile(path string) error {
	j.filePath = path
	j.titleChan <- path
	return j.ReloadFile()
}

func (j *jsEditor) ReloadFile() error {
	contents, err := ioutil.ReadFile(j.filePath)
	if err != nil {
		return err
	}
	j.elem.Call("setContents", string(contents))
	return nil
}

func (j *jsEditor) GetCursor() int {
	return j.elem.Call("getCursorIndex").Int()
}

func (j *jsEditor) SetCursor(i int) error {
	j.elem.Call("setCursorIndex", i)
	return nil
}

func (j *jsEditor) Titles() <-chan string {
	return j.titleChan
}
