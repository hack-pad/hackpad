package main

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/global"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/process"
	"github.com/johnstarich/go-wasm/log"
)

func main() {
	process.Init()
	fs.Init()
	global.Set("dump", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		log.Error("Process: ", process.Dump(), "\n\nFiles: ", fs.Dump())
		return nil
	}))
	interop.SetInitialized()
	select {}
}
