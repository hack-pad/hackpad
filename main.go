//go:build js
// +build js

package main

import (
	"os"
	"runtime/debug"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/worker"
)

type domShim struct {
	dom *worker.DOM
}

func main() {
	defer common.CatchExceptionHandler(func(err error) {
		log.Error("Hackpad panic:", err, "\n", string(debug.Stack()))
		os.Exit(1)
	})

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

	if err := shim.Install("editor"); err != nil {
		panic(err)
	}
	if err := shim.Install("sh"); err != nil {
		panic(err)
	}

	if err := dom.Start(); err != nil {
		panic(err)
	}

	select {}
}
