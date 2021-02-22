package ide

import (
	_ "embed"
	"fmt"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/css"
	"github.com/johnstarich/go-wasm/cmd/editor/element"
)

var (
	//go:embed dropdown.css
	dropdownCSS string
)

func init() {
	css.Add(dropdownCSS)
}

type dropdown struct {
	*element.Element

	attached *element.Element
	opened   bool
}

func newDropdown(attachTo, content *element.Element) *dropdown {
	drop := &dropdown{
		Element:  element.New("div"),
		attached: attachTo,
	}
	element.GetDocument().AddEventListener("click", func(event js.Value) {
		if !event.Call("composedPath").Call("includes", drop).Bool() {
			drop.Close()
		}
	})
	drop.AppendChild(content)
	drop.AddClass("dropdown")
	element.Body().InsertBefore(drop.Element, element.Body().FirstChild())
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
	viewportRect := element.ViewportRect()
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
