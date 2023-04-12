//go:build js
// +build js

package fs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"syscall/js"
	"time"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
	"github.com/machinebox/progress"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/hack-pad/hackpad/internal/promise"
	"github.com/johnstarich/go/datasize"
)

func overlayIndexedDB(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	go func() {
		err := OverlayIndexedDB(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed overlaying IndexedDB FS"))
		} else {
			log.Debug("Successfully overlayed IndexedDB FS")
			resolve(nil)
		}
	}()
	return prom.JSValue()
}

func OverlayIndexedDB(args []js.Value) (err error) {
	if len(args) == 0 {
		return errors.New("overlayIndexedDB: mount path is required")
	}
	mountPath := args[0].String()
	var options map[string]js.Value
	if len(args) >= 2 && args[1].Type() == js.TypeObject {
		options = interop.Entries(args[1])
	}

	durability := idb.DurabilityDefault
	if cacheEnabled, ok := options["cache"]; ok && cacheEnabled.Bool() {
		durability = idb.DurabilityRelaxed
	}

	idbFS, err := indexeddb.NewFS(context.Background(), mountPath, indexeddb.Options{
		TransactionDurability: durability,
	})
	if err != nil {
		return err
	}
	return fs.Overlay(mountPath, idbFS)
}

func overlayTarGzip(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	log.Debug("Backgrounding overlay request")
	go func() {
		err := OverlayTarGzip(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed overlaying .tar.gz FS"))
		} else {
			log.Debug("Successfully overlayed .tar.gz FS")
			resolve(nil)
		}
	}()
	return prom.JSValue()
}

func OverlayTarGzip(args []js.Value) error {
	if len(args) < 2 {
		return errors.New("overlayTarGzip: mount path and .tar.gz URL path is required")
	}

	mountPath := args[0].String()
	downloadPath := args[1].String()
	var options map[string]js.Value
	if len(args) >= 3 && args[2].Type() == js.TypeObject {
		options = interop.Entries(args[2])
	}
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

	reader := resp.Body
	if progressCallback := options["progress"]; progressCallback.Type() == js.TypeFunction && resp.ContentLength > 0 {
		reader = wrapProgress(reader, resp.ContentLength, func(percentage float64) {
			progressCallback.Invoke(percentage)
		})
	}
	persist := options["persist"].Truthy()
	shouldCache := func(string, hackpadfs.FileInfo) bool { return true }
	if options["skipCacheDirs"].Type() == js.TypeObject {
		skipDirs := make(map[string]bool)
		for _, d := range interop.StringsFromJSValue(options["skipCacheDirs"]) {
			skipDirs[common.ResolvePath(process.Current().WorkingDirectory(), d)] = true
		}
		maxFileBytes := datasize.Kibibytes(100).Bytes()
		shouldCache = func(name string, info hackpadfs.FileInfo) bool {
			return !skipDirs[path.Dir(name)] && info.Size() < maxFileBytes
		}
	}
	return fs.OverlayTarGzip(mountPath, reader, persist, shouldCache)
}

func wrapProgress(r io.ReadCloser, contentLength int64, setProgress func(float64)) io.ReadCloser {
	progressR := progress.NewReader(r)
	ctx, wrapper := newReadCloseWrapper(progressR, r)
	go func() {
		progressChan := progress.NewTicker(context.Background(), progressR, contentLength, 100*time.Millisecond)
		for p := range progressChan {
			select {
			case <-ctx.Done():
				setProgress(100)
				return
			default:
				setProgress(p.Percent())
			}
		}
	}()
	return wrapper
}

type readCloseWrapper struct {
	io.Reader
	closer io.Closer
	cancel context.CancelFunc
}

func newReadCloseWrapper(r io.Reader, closer io.Closer) (context.Context, io.ReadCloser) {
	ctx, cancel := context.WithCancel(context.Background())
	return ctx, &readCloseWrapper{
		Reader: r,
		closer: closer,
		cancel: cancel,
	}
}

func (r *readCloseWrapper) Close() error {
	r.cancel()
	return r.closer.Close()
}
