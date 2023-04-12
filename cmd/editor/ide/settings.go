//go:build js
// +build js

package ide

import (
	_ "embed"
	"fmt"
	"os"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/css"
	"github.com/hack-pad/hackpad/cmd/editor/dom"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/promise"
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

func newSettings() *dom.Element {
	button := dom.New("button")
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

func newSettingsDropdown(attachTo *dom.Element) *dropdown {
	elem := dom.New("div")
	elem.SetInnerHTML(settingsHTML)
	elem.AddClass("settings-dropdown")
	drop := newDropdown(attachTo, elem)

	listenButton := func(name, prompt string, fn func()) {
		elem.
			QuerySelector(fmt.Sprintf("button[title=%q]", name)).
			AddEventListener("click", func(event js.Value) {
				if prompt == "" || dom.Confirm(prompt) {
					go fn()
				}
			})
	}

	destroyMount := func(path string) promise.Promise {
		return promise.From(global.Get("destroyMount").Invoke(path))
	}
	listenButton("reset", "Erase all data and reload?", func() {
		mounts := interop.StringsFromJSValue(global.Get("getMounts").Invoke())
		var promises []promise.Promise
		for _, mount := range mounts {
			promises = append(promises, destroyMount(mount))
		}
		for _, p := range promises {
			_, _ = p.Await()
		}
		dom.Reload()
	})
	listenButton("clean build cache", "", func() {
		cache, err := os.UserCacheDir()
		if err == nil {
			destroyMount(cache)
		}
	})
	listenButton("reload programs", "Reinstall programs and reload?", func() {
		_, _ = destroyMount("/bin").Await()
		_, _ = destroyMount(goInstallPath).Await()
		dom.Reload()
	})
	return drop
}
