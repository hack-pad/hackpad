// +build js

package main

import (
	"path/filepath"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/global"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/js/fs"
	"github.com/johnstarich/go-wasm/internal/js/process"
	libProcess "github.com/johnstarich/go-wasm/internal/process"
	"github.com/johnstarich/go-wasm/internal/terminal"
	"github.com/johnstarich/go-wasm/log"
)

func main() {
	process.Init()
	fs.Init()
	global.Set("spawnTerminal", js.FuncOf(terminal.SpawnTerminal))
	global.Set("dump", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go func() {
			basePath := ""
			if len(args) >= 1 {
				basePath = args[0].String()
				if filepath.IsAbs(basePath) {
					basePath = filepath.Clean(basePath)
				} else {
					basePath = filepath.Join(libProcess.Current().WorkingDirectory(), basePath)
				}
			}
			var fsDump interface{}
			if basePath != "" {
				fsDump = fs.Dump(basePath)
			}
			log.Error("Process:\n", process.Dump(), "\n\nFiles:\n", fsDump)
		}()
		return nil
	}))
	global.Set("profile", js.FuncOf(interop.ProfileJS))
	global.Set("install", js.FuncOf(installFunc))
	interop.SetInitialized()
	select {}
}
