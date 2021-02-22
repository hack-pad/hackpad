// +build js

package ide

import (
	_ "embed"
	"fmt"
	"os"
	"syscall/js"

	"github.com/johnstarich/go-wasm/cmd/editor/css"
	"github.com/johnstarich/go-wasm/cmd/editor/element"
	"github.com/johnstarich/go-wasm/internal/global"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
)

var (
	//go:embed settings.html
	settingsHTML string
	//go:embed settings.css
	settingsCSS string
)

const (
	goInstallPath = "/usr/local/go"
)

func init() {
	css.Add(settingsCSS)
}

func newSettings() *element.Element {
	button := element.New("button")
	button.SetInnerHTML(`<span class="fa fa-cog"></span>`)
	button.SetAttribute("className", "control")
	button.SetAttribute("title", "Settings")

	drop := newSettingsDropdown(button)
	button.AddEventListener("click", func(event js.Value) {
		event.Call("stopPropagation")
		drop.Toggle()
	})
	return button
}

func newSettingsDropdown(attachTo *element.Element) *dropdown {
	elem := element.New("div")
	elem.SetInnerHTML(settingsHTML)
	elem.AddClass("settings-dropdown")
	drop := newDropdown(attachTo, elem)

	listenButton := func(name string, fn func()) {
		elem.
			QuerySelector(fmt.Sprintf("button[title=%q]", name)).
			AddEventListener("click", func(event js.Value) {
				go fn()
			})
	}

	destroyMount := func(path string) promise.Promise {
		return promise.From(global.Get("destroyMount").Invoke(path))
	}
	listenButton("reset", func() {
		mounts := interop.Keys(global.Get("getMounts").Invoke())
		var promises []promise.Promise
		for _, mount := range mounts {
			promises = append(promises, destroyMount(mount))
		}
		for _, p := range promises {
			_, _ = p.Await()
		}
		js.Global().Get("window").Get("location").Call("reload")
	})
	listenButton("clean build cache", func() {
		cache, err := os.UserCacheDir()
		if err == nil {
			destroyMount(cache)
		}
	})
	listenButton("reload go install", func() {
		_, _ = destroyMount(goInstallPath).Await()
		js.Global().Get("window").Get("location").Call("reload")
	})
	return drop
}
