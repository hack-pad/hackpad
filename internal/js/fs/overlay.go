// +build js

package fs

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"syscall/js"
	"time"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb"
	"github.com/machinebox/progress"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
)

func overlayZip(this js.Value, args []js.Value) interface{} {
	resolve, reject, prom := promise.New()
	log.Debug("Backgrounding overlay request")
	go func() {
		err := OverlayZip(args)
		if err != nil {
			reject(interop.WrapAsJSError(err, "Failed overlaying zip FS"))
		} else {
			log.Debug("Successfully overlayed zip FS")
			resolve(nil)
		}
	}()
	return prom
}

func OverlayZip(args []js.Value) error {
	if len(args) != 2 {
		return errors.New("overlayZip: mount path and zip URL path is required")
	}

	mountPath := args[0].String()
	zipPath := args[1].String()
	log.Debug("Downloading overlay zip FS: ", zipPath)
	u, err := url.Parse(zipPath)
	if err != nil {
		return err
	}
	resp, err := http.Get(u.Path) // only download from current server, not just any URL
	if err != nil {
		return err
	}
	log.Debug("Download response received. Reading body...")

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug("Finished reading download data. Overlaying FS...")
	if err := resp.Body.Close(); err != nil {
		return err
	}

	z, err := zip.NewReader(bytes.NewReader(body), resp.ContentLength)
	if err != nil {
		return err
	}
	return fs.Overlay(mountPath, &zipFS{zipfs.New(z)})
}

type zipFS struct {
	afero.Fs
}

func (z *zipFS) Open(name string) (hackpadfs.File, error) {
	return z.Fs.Open(name)
}

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
	return prom
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

	//shouldCache := func(string) bool { return false }
	if cacheEnabled, ok := options["cacheInfo"]; ok && cacheEnabled.Bool() {
		panic("cache not implemented")
		//shouldCache = func(string) bool { return true }
	}

	idbFS, err := indexeddb.NewFS(context.Background(), mountPath, idb.Global())
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
	return prom
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
	return fs.OverlayTarGzip(mountPath, reader, persist)
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
