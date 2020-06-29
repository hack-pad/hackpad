package main

import (
	"io"
	"syscall/js"
)

type elementWriter struct {
	element js.Value
}

func newElementWriter(elem js.Value) io.Writer {
	return &elementWriter{
		element: elem,
	}
}

func (w *elementWriter) Write(p []byte) (n int, err error) {
	textNode := document.Call("createTextNode", string(p))
	w.element.Call("appendChild", textNode)
	return len(p), nil
}
