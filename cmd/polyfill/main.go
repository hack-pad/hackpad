package main

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/global"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/js/fs"
	"github.com/johnstarich/go-wasm/internal/js/process"
	"github.com/johnstarich/go-wasm/log"
)

func main() {
	process.Init()
	fs.Init()
	global.Set("dump", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Error("Process:\n", process.Dump(), "\n\nFiles:\n", fs.Dump())
		return nil
	}))
	interop.SetInitialized()
	select {}
}
