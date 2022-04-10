//go:build js
// +build js

package main

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime/debug"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsfunc"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/terminal"
	"github.com/hack-pad/hackpad/internal/worker"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
	"github.com/johnstarich/go/datasize"
)

type domShim struct {
	dom *worker.DOM
}

func main() {
	defer common.CatchExceptionHandler(func(err error) {
		log.Error("Hackpad panic:", err, "\n", string(debug.Stack()))
		os.Exit(1)
	})

	bootCtx := context.Background()
	//bootCtx, bootCancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer bootCancel()
	dom, err := worker.ExecDOM(
		bootCtx,
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
	global.Set("profile", jsfunc.NonBlocking(interop.ProfileJS))
	global.Set("install", jsfunc.Promise(shim.installFunc))
	global.Set("spawnTerminal", jsfunc.NonBlocking(terminal.SpawnTerminal))

	if err := setUpFS(shim); err != nil {
		panic(err)
	}

	if err := dom.Start(); err != nil {
		panic(err)
	}

	select {}
}

func setUpFS(shim domShim) error {
	const dirPerm = 0700
	if err := os.MkdirAll("/bin", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/bin", idb.DurabilityRelaxed); err != nil {
		return err
	}
	if err := overlayIndexedDB("/home/me", idb.DurabilityDefault); err != nil {
		return err
	}
	if err := os.MkdirAll("/home/me/.cache", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/home/me/.cache", idb.DurabilityRelaxed); err != nil {
		return err
	}
	if err := os.MkdirAll("/usr/local/go", dirPerm); err != nil {
		return err
	}
	if err := overlayTarGzip("/usr/local/go", "wasm/go.tar.gz", []string{
		"/usr/local/go/bin/js_wasm",
		"/usr/local/go/pkg/tool/js_wasm",
	}); err != nil {
		return err
	}

	if err := shim.Install("editor"); err != nil {
		return err
	}
	if err := shim.Install("sh"); err != nil {
		return err
	}
	return nil
}

func overlayIndexedDB(mountPath string, durability idb.TransactionDurability) error {
	idbFS, err := indexeddb.NewFS(context.Background(), mountPath, indexeddb.Options{
		TransactionDurability: durability,
	})
	if err != nil {
		return err
	}
	return fs.Overlay(mountPath, idbFS)
}

func overlayTarGzip(mountPath, downloadPath string, skipCacheDirs []string) error {
	log.Debug("Downloading overlay .tar.gz FS: ", downloadPath)
	u, err := url.Parse(downloadPath)
	if err != nil {
		return err
	}
	// only download from current server, not just any URL
	resp, err := http.Get(u.Path) // nolint:bodyclose // Body is closed in OverlayTarGzip handler to keep this async
	if err != nil {
		return err
	}
	log.Debug("Download response received. Reading body...")

	skipDirs := make(map[string]bool)
	for _, d := range skipCacheDirs {
		skipDirs[common.ResolvePath("/", d)] = true
	}
	maxFileBytes := datasize.Kibibytes(100).Bytes()
	shouldCache := func(name string, info hackpadfs.FileInfo) bool {
		return !skipDirs[path.Dir(name)] && info.Size() < maxFileBytes
	}
	return fs.OverlayTarGzip(mountPath, resp.Body, true, shouldCache)
}
