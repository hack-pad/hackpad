// +build js

package ide

import "github.com/johnstarich/go-wasm/cmd/editor/element"

type EditorBuilder interface {
	New(elem *element.Element) Editor
}

type Editor interface {
	OpenFile(path string) error
	CurrentFile() string
	ReloadFile() error
	GetCursor() int
	SetCursor(i int) error
	Titles() <-chan string
}
