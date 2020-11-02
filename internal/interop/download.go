// +build js

package interop

import "syscall/js"

var (
	jsBlob     = js.Global().Get("Blob")
	jsDocument = js.Global().Get("document")
	jsURL      = js.Global().Get("URL")
)

func StartDownload(contentType, fileName string, b []byte) {
	jsBuf := NewByteArray(b)
	blob := jsBlob.New([]interface{}{jsBuf}, map[string]interface{}{
		"type": contentType,
	})
	link := jsDocument.Call("createElement", "a")
	link.Set("href", jsURL.Call("createObjectURL", blob))
	link.Set("download", fileName)
	link.Call("click")
}
