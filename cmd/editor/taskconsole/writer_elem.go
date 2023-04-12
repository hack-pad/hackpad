//go:build js
// +build js

package taskconsole

import (
	"io"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

const (
	maxJSInt = (1 << 53) - 1
)

type elementWriter struct {
	element *dom.Element
	class   string
}

func newElementWriter(elem *dom.Element, class string) interface {
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
	textNode := dom.New("span")
	w.element.AppendChild(textNode)
	if w.class != "" {
		textNode.AddClass(w.class)
	}
	textNode.SetInnerText(s)
	w.element.SetScrollTop(maxJSInt)
	return len(s), nil
}
