module github.com/johnstarich/go-wasm

go 1.16

require (
	github.com/avct/uasurfer v0.0.0-20191028135549-26b5daa857f1
	github.com/fatih/color v1.9.0
	github.com/hack-pad/go-indexeddb v0.0.0-20210627055059-75ac2a19af43
	github.com/hack-pad/hackpadfs v0.0.0-20210721055107-9de55e3d77dd
	github.com/johnstarich/go/datasize v0.0.1
	github.com/machinebox/progress v0.2.0
	github.com/mattn/go-tty v0.0.3
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.3.0
	github.com/stretchr/testify v1.5.1
	go.uber.org/atomic v1.6.0
	mvdan.cc/sh/v3 v3.1.2
)

replace github.com/spf13/afero v1.3.0 => github.com/johnstarich/afero v1.3.2-0.20210214021553-81c4e4e83b19
