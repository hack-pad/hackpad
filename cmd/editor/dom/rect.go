//go:build js
// +build js

package dom

import "syscall/js"

type Rect struct {
	Left, Top, Right, Bottom float64
	Width, Height            float64
}

func newRect(domRect js.Value) *Rect {
	return &Rect{
		Left:   domRect.Get("left").Float(),
		Top:    domRect.Get("top").Float(),
		Right:  domRect.Get("right").Float(),
		Bottom: domRect.Get("bottom").Float(),
		Width:  domRect.Get("width").Float(),
		Height: domRect.Get("height").Float(),
	}
}
