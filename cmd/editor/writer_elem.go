package main

import (
	"io"
	"syscall/js"
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
	textNode.Call("scrollIntoView", false)
	return len(s), nil
}
