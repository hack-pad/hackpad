module github.com/hack-pad/hackpad

go 1.18

replace github.com/hack-pad/hackpadfs => github.com/paralin/hackpadfs v0.0.0-20220810055416-aba65e99a16a // fix-go1.19

require (
	github.com/avct/uasurfer v0.0.0-20191028135549-26b5daa857f1
	github.com/hack-pad/go-indexeddb v0.2.1-0.20220430204450-f0f0319256f1
	github.com/hack-pad/hackpadfs v0.1.5
	github.com/hack-pad/hush v0.1.0
	github.com/johnstarich/go/datasize v0.0.1
	github.com/machinebox/progress v0.2.0
	github.com/pkg/errors v0.9.1
	go.uber.org/atomic v1.6.0
)

require (
	github.com/fatih/color v1.12.0 // indirect
	github.com/matryer/is v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-tty v0.0.3 // indirect
	golang.org/x/sys v0.0.0-20210503080704-8803ae5d1324 // indirect
	mvdan.cc/sh/v3 v3.3.0 // indirect
)
