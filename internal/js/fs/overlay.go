package fs

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"syscall/js"

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
	return fs.OverlayZip(mountPath, z)
}

func overlayStorage(this js.Value, args []js.Value) interface{} {
	if len(args) != 2 {
		return errors.New("overlayStorage: mount path and storer value (i.e. localStorage) are required")
	}

	mountPath := args[0].String()
	jsStorer := args[1]
	err := fs.OverlayStorage(mountPath, fs.NewJSStorage(jsStorer))
	if err != nil {
		log.Error("Failed to overlay storage FS:", err)
	}
	return nil
}
