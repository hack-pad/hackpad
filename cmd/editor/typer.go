package main

import (
	"strings"
	"syscall/js"
	"unicode"
)

// codeTyper is fired on keydown event
func codeTyper(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		event := args[0]
		handleKeydown(event)
	}
	return nil
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
	metaKey := event.Get("metaKey").Bool()

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
		if metaKey {
			preventDefault()
			runPlayground()
			return
		}

		lastNewLine := strings.LastIndexByte(text[:selectionStart], '\n')
		if lastNewLine != -1 {
			leadingChars := text[lastNewLine+1 : selectionStart]
			leadingSpace := leadingChars[:len(leadingChars)-len(strings.TrimLeftFunc(leadingChars, unicode.IsSpace))]
			prevChar := string(leadingChars[len(leadingChars)-1])
			nextChar := text[selectionStart : selectionStart+1]

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
		prevChar := text[selectionStart-1 : selectionStart]
		nextChar := text[selectionStart : selectionStart+1]
		if parseBracket(prevChar).next == nextChar {
			document.Call("execCommand", "forwardDelete", false)
		}
		return
	}

	if selectionStart != selectionEnd {
		return
	}

	closer := parseBracket(key).next
	afterSelection := text[selectionStart : selectionStart+1] // TODO handle end case
	if closer != "" && afterSelection != closer {
		preventDefault()
		insertText(key + closer)
		target.Set("selectionStart", selectionStart+1)
		target.Set("selectionEnd", selectionStart+1)
		return
	}

	nextChar := text[selectionStart : selectionStart+1]
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
