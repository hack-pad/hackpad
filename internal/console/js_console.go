//go:build js && wasm
// +build js,wasm

package console

import (
	"io"
	"syscall/js"
)

type console struct {
	stdout, stderr, note io.Writer
}

func New(element js.Value) Console {
	element.Set("innerHTML", `
<pre class="console-output"></pre>
`)
	element.Get("classList").Call("add", "console")
	outputElem := element.Call("querySelector", ".console-output")
	return &console{
		stdout: newElementWriter(outputElem, ""),
		stderr: newElementWriter(outputElem, "stderr"),
		note:   newElementWriter(outputElem, "note"),
	}
}

func (c *console) Stdout() io.Writer { return c.stdout }
func (c *console) Stderr() io.Writer { return c.stderr }
func (c *console) Note() io.Writer   { return c.note }
