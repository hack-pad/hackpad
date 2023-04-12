//go:build js
// +build js

package css

import (
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

var sheet = createStyleSheet()

func createStyleSheet() js.Value {
	style := dom.New("style")
	style.SetAttribute("type", "text/css")
	dom.Head().AppendChild(style)
	return style.GetProperty("sheet")
}

func Add(css string) {
	sheet.Call("insertRule", fmt.Sprintf(`@media all { %s }`, css))
}
