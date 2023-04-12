//go:build js
// +build js

package dom

import (
	"runtime/debug"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
)

type Element struct {
	elem js.Value
}

type EventListener = func(event js.Value)

func New(tag string) *Element {
	return document.CreateElement(tag)
}

func NewFromJS(elem js.Value) *Element {
	if elem.IsNull() {
		return nil
	}
	return &Element{elem}
}

func (e *Element) JSValue() js.Value {
	return e.elem
}

func (e *Element) GetProperty(property string) js.Value {
	return e.elem.Get(property)
}

func (e *Element) SetProperty(property string, value js.Value) {
	e.elem.Set(property, value)
}

func (e *Element) AppendChild(child *Element) {
	e.elem.Call("appendChild", child.elem)
}

func (e *Element) InsertBefore(newChild, referenceNode *Element) {
	e.elem.Call("insertBefore", newChild.elem, referenceNode.elem)
}

func (e *Element) FirstChild() *Element {
	child := e.elem.Get("firstChild")
	if child.IsNull() {
		return nil
	}
	return NewFromJS(child)
}

func (e *Element) SetInnerHTML(contents string) {
	e.elem.Set("innerHTML", contents)
}

func (e *Element) SetInnerText(contents string) {
	e.elem.Set("innerText", contents)
}

func (e *Element) InnerText() string {
	return e.elem.Get("innerText").String()
}

func (e *Element) AddClass(class string) {
	e.elem.Get("classList").Call("add", class)
}

func (e *Element) RemoveClass(class string) {
	e.elem.Get("classList").Call("remove", class)
}

func (e *Element) ToggleClass(class string) {
	e.elem.Get("classList").Call("toggle", class)
}

func (e *Element) SetStyle(props map[string]interface{}) {
	style := e.elem.Get("style")
	for prop, value := range props {
		style.Set(prop, value)
	}
}

func (e *Element) SetAttribute(prop, value string) {
	e.elem.Set(prop, value)
}

func (e *Element) GetBoundingClientRect() *Rect {
	return newRect(e.elem.Call("getBoundingClientRect"))
}

func (e *Element) QuerySelector(query string) *Element {
	return NewFromJS(e.elem.Call("querySelector", query))
}

func sliceFromArray(array js.Value) []*Element {
	var elements []*Element
	for _, node := range interop.SliceFromJSValue(array) {
		elements = append(elements, NewFromJS(node))
	}
	return elements
}

func (e *Element) QuerySelectorAll(query string) []*Element {
	return sliceFromArray(e.elem.Call("querySelectorAll", query))
}

func (e *Element) RemoveEventListener(name string, listener js.Func) {
	e.elem.Call("removeEventListener", name, listener)
}

func (e *Element) AddEventListener(name string, listener EventListener) js.Func {
	listenerFunc := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		defer common.CatchExceptionHandler(func(err error) {
			log.Error("recovered from panic: ", err, "\n", string(debug.Stack()))
		})
		listener(args[0])
		return nil
	})
	e.elem.Call("addEventListener", name, listenerFunc)
	return listenerFunc
}

func (e *Element) Focus() {
	e.elem.Call("focus")
}

func (e *Element) Children() []*Element {
	return sliceFromArray(e.elem.Get("children"))
}

func (e *Element) Remove() {
	e.elem.Call("remove")
}

func (e *Element) Value() string {
	return e.elem.Get("value").String()
}

func (e *Element) SetValue(value string) {
	e.elem.Set("value", value)
}

func (e *Element) SetScrollTop(pixels int) {
	e.elem.Set("scrollTop", pixels)
}
