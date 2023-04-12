//go:build js
// +build js

package dom

import "syscall/js"

var (
	document = &Document{NewFromJS(js.Global().Get("document"))}
	body     = NewFromJS(document.elem.Get("body"))
	head     = NewFromJS(document.elem.Get("head"))
)

type Document struct {
	*Element
}

func GetDocument() *Document {
	return document
}

func Body() *Element {
	return body
}

func Head() *Element {
	return head
}

func (d *Document) GetElementByID(id string) *Element {
	return NewFromJS(d.elem.Call("getElementById", id))
}

func (d *Document) Body() *Element {
	return body
}

func (d *Document) Head() *Element {
	return head
}

func (d *Document) CreateElement(tag string) *Element {
	return NewFromJS(d.elem.Call("createElement", tag))
}
