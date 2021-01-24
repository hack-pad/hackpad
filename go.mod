module github.com/johnstarich/go-wasm

go 1.13

require (
	github.com/avct/uasurfer v0.0.0-20191028135549-26b5daa857f1
	github.com/fatih/color v1.9.0
	github.com/johnstarich/go/datasize v0.0.1
	github.com/machinebox/progress v0.2.0
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/nsf/termbox-go v0.0.0-20200418040025-38ba6e5628f1
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.3.0
	github.com/stretchr/testify v1.5.1
	go.uber.org/atomic v1.6.0
	mvdan.cc/sh/v3 v3.1.2
)

replace github.com/spf13/afero v1.3.0 => github.com/johnstarich/afero v1.3.2-0.20200824034706-e0c81fb79d7b
