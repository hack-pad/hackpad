package main

import (
	"io"
	"unicode/utf8"
)

type runeReader struct {
	io.Reader
}

func newRuneReader(r io.Reader) io.RuneReader {
	return &runeReader{r}
}

func (b *runeReader) ReadRune() (r rune, n int, err error) {
	oneByte := make([]byte, 1)
	var buf []byte
	for !utf8.FullRune(buf) {
		n, err = b.Read(oneByte)
		if err != nil {
			r = utf8.RuneError
			return
		}
		buf = append(buf, oneByte[0])
	}
	r, n = utf8.DecodeRune(buf)
	return
}
