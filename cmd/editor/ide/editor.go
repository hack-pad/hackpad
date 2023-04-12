//go:build js
// +build js

package ide

import "github.com/hack-pad/hackpad/cmd/editor/dom"

type EditorBuilder interface {
	New(elem *dom.Element) Editor
}

type Editor interface {
	OpenFile(path string) error
	CurrentFile() string
	ReloadFile() error
	GetCursor() int
	SetCursor(i int) error
	Titles() <-chan string
}
