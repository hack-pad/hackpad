package ide

import "syscall/js"

type EditorBuilder interface {
	New(elem js.Value) Editor
}

type Editor interface {
	OpenFile(path string) error
	ReloadFile() error
	GetCursor() int
	SetCursor(i int) error
}
