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

func ViewportRect() *Rect {
	window := js.Global()
	width, height := window.Get("innerWidth").Float(), window.Get("innerHeight").Float()
	return &Rect{
		Left:   0,
		Top:    0,
		Right:  width,
		Bottom: height,
		Width:  width,
		Height: height,
	}
}
