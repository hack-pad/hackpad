//go:build js
// +build js

package plaineditor

import (
	"runtime/debug"
	"strings"
	"syscall/js"
	"unicode"

	"github.com/hack-pad/hackpad/internal/log"
)

var (
	document = js.Global().Get("document")
)

// codeTyper is fired on keydown event
func codeTyper(event js.Value) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		log.Error("Failed to handle keydown:", r, "\n"+string(debug.Stack()))
	}()

	handleKeydown(event)
}

const (
	KeyTab       = "Tab"
	KeyEnter     = "Enter"
	KeyBackspace = "Backspace"
)

func handleKeydown(event js.Value) {
	target := event.Get("target")
	text := target.Get("value").String()
	selectionStart := target.Get("selectionStart").Int()
	selectionEnd := target.Get("selectionEnd").Int()
	key := event.Get("key").String()
	code := event.Get("code").String()
	// metaKey := event.Get("metaKey").Bool()

	preventDefault := func() {
		event.Call("preventDefault")
	}
	insertText := func(s string) {
		document.Call("execCommand", "insertText", false, s)
	}

	if code == KeyTab {
		preventDefault()
		insertText("\t")
		return
	}

	if code == KeyEnter {
		// TODO restore cmd+enter triggering run button
		// if metaKey {
		//     preventDefault()
		//     runPlayground()
		//     return
		// }

		lastNewLine := strings.LastIndexByte(slice(text, 0, selectionStart), '\n')
		if lastNewLine != -1 {
			leadingChars := slice(text, lastNewLine+1, selectionStart)
			leadingSpace := slice(leadingChars, 0, len(leadingChars)-len(strings.TrimLeftFunc(leadingChars, unicode.IsSpace)))
			prevChar := slice(leadingChars, -1, 0)
			nextChar := slice(text, selectionStart, selectionStart+1)

			newLinePrefix := "\n" + leadingSpace
			newLineSuffix := ""
			prevBracket := parseBracket(prevChar)
			nextBracket := parseBracket(nextChar)
			if prevBracket.opener {
				newLinePrefix += "\t"
				if nextBracket.closer {
					newLineSuffix += "\n" + leadingSpace
				}
			}
			insertText(newLinePrefix + newLineSuffix)
			target.Set("selectionStart", selectionStart+len(newLinePrefix))
			target.Set("selectionEnd", selectionStart+len(newLinePrefix))
			preventDefault()
		}
		return
	}

	if code == KeyBackspace {
		prevChar := slice(text, selectionStart-1, selectionStart)
		nextChar := slice(text, selectionStart, selectionStart+1)
		if parseBracket(prevChar).next == nextChar {
			document.Call("execCommand", "forwardDelete", false)
		}
		return
	}

	if selectionStart != selectionEnd {
		return
	}

	closer := parseBracket(key).next
	afterSelection := slice(text, selectionStart, selectionStart+1)
	if closer != "" && afterSelection != closer {
		preventDefault()
		insertText(key + closer)
		target.Set("selectionStart", selectionStart+1)
		target.Set("selectionEnd", selectionStart+1)
		return
	}

	nextChar := slice(text, selectionStart, selectionStart+1)
	if key == nextChar && parseBracket(nextChar).closer {
		preventDefault()
		target.Set("selectionStart", selectionStart+1)
		target.Set("selectionEnd", selectionStart+1)
		return
	}
}

type Bracket struct {
	opener bool
	closer bool
	next   string
}

func parseBracket(s string) Bracket {
	switch s {
	case "{":
		return Bracket{opener: true, closer: false, next: "}"}
	case "}":
		return Bracket{opener: false, closer: true, next: ""}
	case "[":
		return Bracket{opener: true, closer: false, next: "]"}
	case "]":
		return Bracket{opener: false, closer: true, next: ""}
	case "(":
		return Bracket{opener: true, closer: false, next: ")"}
	case ")":
		return Bracket{opener: false, closer: true, next: ""}
	case `"`:
		return Bracket{opener: true, closer: true, next: `"`}
	case "'":
		return Bracket{opener: true, closer: true, next: "'"}
	default:
		return Bracket{next: ""}
	}
}

func slice(s string, start, end int) string {
	if start < 0 {
		start += len(s)
		if start < 0 {
			start = 0
		}
	}
	if start > len(s) {
		start = len(s) - 1
	}
	if end < start || end > len(s) {
		end = len(s)
	}
	return s[start:end]
}
