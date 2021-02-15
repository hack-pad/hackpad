package fs

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

func TestEncoding(t *testing.T) {
	for _, tc := range []string{
		"",
		"/",
		"?",
		"some CRAZY file name, with punctuation?!",
		"👍",
	} {
		t.Run(tc, func(t *testing.T) {
			name := pathToName(tc)
			for _, r := range name {
				assert.False(t, unicode.IsUpper(r), r)
				assert.NotEqual(t, '/', r, r)
			}
			path, err := nameToPath(name)
			assert.NoError(t, err)
			assert.Equal(t, tc, path)
		})
	}
}
