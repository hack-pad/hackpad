//go:build js
// +build js

package main

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/worker"
)

type domShim struct {
	dom *worker.DOM
}

func main() {
	dom, err := worker.ExecDOM(
		jsworker.GetLocal(),
		"editor",
		[]string{"-editor=editor"},
		"/home/me",
		map[string]string{
			"GOMODCACHE": "/home/me/.cache/go-mod",
			"GOPROXY":    "https://proxy.golang.org/",
			"GOROOT":     "/usr/local/go",
			"HOME":       "/home/me",
			"PATH":       "/bin:/home/me/go/bin:/usr/local/go/bin/js_wasm:/usr/local/go/pkg/tool/js_wasm",
		},
	)
	if err != nil {
		panic(err)
	}

	shim := domShim{dom}
	global.Set("profile", js.FuncOf(interop.ProfileJS))
	global.Set("install", js.FuncOf(shim.installFunc))
	//global.Set("spawnTerminal", js.FuncOf(terminal.SpawnTerminal))
	select {}
}
