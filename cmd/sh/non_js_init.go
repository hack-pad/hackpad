// +build !js

package main

import "github.com/nsf/termbox-go"

func init() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	termbox.SetCursor(0, 0)
	if err := termbox.Flush(); err != nil {
		panic(err)
	}
}
