package css

import (
	"fmt"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/element"
)

var sheet = createStyleSheet()

func createStyleSheet() js.Value {
	style := element.New("style")
	style.SetAttribute("type", "text/css")
	element.Head().AppendChild(style)
	return style.GetProperty("sheet")
}

func Add(css string) {
	sheet.Call("insertRule", fmt.Sprintf(`@media all { %s }`, css))
}
