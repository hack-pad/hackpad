package fs

import (
	"encoding/base64"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

/*
To decode all paths in JS, use this:

(await nativeIO.getAll()).map(f => f.replace(/_a([a-z])/g, m => m.slice(2).toUpperCase())).map(atob)
*/

var nameEncoding = base64.RawStdEncoding

const (
	// std encoding contains '/' and uppercase letters, so replace these with underscore and lowercase letter variants
	encodingPrefix      = '_'
	upperLetterEncoding = 'a'
	slashEncoding       = 'b'
	plusEncoding        = 'c'
)

func pathToName(path string) string {
	var sb strings.Builder
	for _, r := range nameEncoding.EncodeToString([]byte(path)) {
		switch {
		case unicode.IsLetter(r) && unicode.IsUpper(r):
			sb.WriteRune(encodingPrefix)
			sb.WriteRune(upperLetterEncoding)
			sb.WriteRune(unicode.ToLower(r))
		case r == '/':
			sb.WriteRune(encodingPrefix)
			sb.WriteRune(slashEncoding)
		case r == '+':
			sb.WriteRune(encodingPrefix)
			sb.WriteRune(plusEncoding)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func nameToPath(name string) (string, error) {
	var sb strings.Builder
	var nextEncoded bool
	var encodeType rune
	for _, r := range name {
		switch {
		case nextEncoded:
			switch encodeType {
			case upperLetterEncoding:
				sb.WriteRune(unicode.ToUpper(r))
				encodeType = rune(0)
				nextEncoded = false
			case rune(0):
				encodeType = r
				switch encodeType {
				case upperLetterEncoding:
					continue // process next rune immediately, don't end encode check
				case slashEncoding:
					sb.WriteRune('/')
				case plusEncoding:
					sb.WriteRune('+')
				default:
					return "", errors.Errorf("Unrecognized underscore-encoded char: %v", r)
				}
				encodeType = rune(0)
				nextEncoded = false
			}
		case r == encodingPrefix:
			nextEncoded = true
		default:
			sb.WriteRune(r)
		}
	}
	name = sb.String()

	b, err := nameEncoding.DecodeString(name)
	return string(b), err
}
