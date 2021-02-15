// +build js

package nativeio

import (
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

var jsNativeIO = js.Global().Get("nativeIO")

func Supported() bool {
	return jsNativeIO.Truthy()
}

type Manager struct {
	jsManager js.Value
}

func New() (*Manager, error) {
	if !Supported() {
		return nil, os.ErrPermission
	}
	return &Manager{jsManager: jsNativeIO}, nil
}

func (m *Manager) OpenOrCreate(name string) (f *File, err error) {
	// TODO remove retry loop once we figure out what causes sporadic 'open() failed' errors
	const maxAttempts = 100
	for i := 0; i < maxAttempts; i++ {
		f, err = m.openOrCreateOnce(name)
		if err == nil {
			return
		}
	}
	return
}

func (m *Manager) openOrCreateOnce(name string) (_ *File, err error) {
	// TODO this open behavior is confusing. I need to know when a file does not exist to return the appropriate error to the user application.
	defer func() {
		common.CatchException(&err)
		if err != nil {
			log.Print("Failed to open or create file '", name, "': ", err)
		}
	}()

	jsFile, err := promise.From(m.jsManager.Call("open", name)).Await()
	return newFile(name, jsFile.(js.Value)), err
}

func (m *Manager) Remove(name string) (err error) {
	defer common.CatchException(&err)
	_, err = promise.From(m.jsManager.Call("delete", name)).Await()
	return err
}

func (m *Manager) GetAll() (_ []string, err error) {
	defer common.CatchException(&err)
	files, err := promise.From(m.jsManager.Call("getAll")).Await()
	return interop.StringsFromJSValue(files.(js.Value)), err
}

func (m *Manager) Rename(oldName, newName string) (err error) {
	defer common.CatchException(&err)
	// TODO renaming to the same name returns: DOMException rename() failed
	_, err = promise.From(m.jsManager.Call("rename", oldName, newName)).Await()
	return err
}
