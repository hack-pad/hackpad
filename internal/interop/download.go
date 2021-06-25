// +build js

package interop

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/blob"
)

var (
	jsBlob     = js.Global().Get("Blob")
	jsDocument = js.Global().Get("document")
	jsURL      = js.Global().Get("URL")
)

func StartDownload(contentType, fileName string, buf []byte) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	b := blob.NewFromBytes(buf)
	blobInstance := jsBlob.New([]interface{}{b}, map[string]interface{}{
		"type": contentType,
	})
	link := jsDocument.Call("createElement", "a")
	link.Set("href", jsURL.Call("createObjectURL", blobInstance))
	link.Set("download", fileName)
	link.Call("click")
}
