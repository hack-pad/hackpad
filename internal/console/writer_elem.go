//go:build js && wasm
// +build js,wasm

package console

import (
	"io"
	"syscall/js"
)

var (
	document = js.Global().Get("document")
)

const (
	maxJSInt = (1 << 53) - 1
)

type elementWriter struct {
	element js.Value
	class   string
}

func newElementWriter(elem js.Value, class string) interface {
	io.Writer
	io.StringWriter
} {
	return &elementWriter{
		element: elem,
		class:   class,
	}
}

func (w *elementWriter) Write(p []byte) (n int, err error) {
	return w.WriteString(string(p))
}

func (w *elementWriter) WriteString(s string) (n int, err error) {
	textNode := document.Call("createElement", "span")
	w.element.Call("appendChild", textNode)
	if w.class != "" {
		textNode.Get("classList").Call("add", w.class)
	}
	textNode.Set("innerText", s)
	w.element.Set("scrollTop", maxJSInt)
	return len(s), nil
}
