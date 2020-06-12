package main

import (
	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/process"
)

func main() {
	process.Init()
	fs.Init()
	interop.SetInitialized()
	select {}
}
