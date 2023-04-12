//go:build js
// +build js

package ide

import (
	_ "embed"
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/css"
	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

var (
	//go:embed dropdown.css
	dropdownCSS string
)

func init() {
	css.Add(dropdownCSS)
}

type dropdown struct {
	*dom.Element

	attached *dom.Element
	opened   bool
}

func newDropdown(attachTo, content *dom.Element) *dropdown {
	drop := &dropdown{
		Element:  dom.New("div"),
		attached: attachTo,
	}
	dom.GetDocument().AddEventListener("click", func(event js.Value) {
		if !event.Call("composedPath").Call("includes", drop.JSValue()).Bool() {
			drop.Close()
		}
	})
	drop.AppendChild(content)
	drop.AddClass("dropdown")
	dom.Body().InsertBefore(drop.Element, dom.Body().FirstChild())
	return drop
}

func (d *dropdown) Toggle() {
	if d.opened {
		d.Close()
	} else {
		d.Open()
	}
}

func (d *dropdown) Open() {
	if d.opened {
		return
	}
	d.opened = true
	rect := d.attached.GetBoundingClientRect()
	viewportRect := dom.ViewportRect()
	top := px(rect.Bottom)
	if rect.Left+rect.Width/2 > viewportRect.Left+viewportRect.Right/2 {
		// on the right half of the screen, align right
		d.SetStyle(map[string]interface{}{
			"top":   top,
			"right": px(viewportRect.Right - rect.Right),
		})
	} else {
		// on the left half of the screen, align left
		d.SetStyle(map[string]interface{}{
			"top":  top,
			"left": px(rect.Left),
		})
	}
	d.AddClass("dropdown-visible")
}

func px(f float64) string {
	return fmt.Sprintf("%fpx", f)
}

func (d *dropdown) Close() {
	if !d.opened {
		return
	}
	d.opened = false
	d.RemoveClass("dropdown-visible")
}
