package main

import (
	"bytes"
	"io"
	"os"
)

type carriageReturnWriter struct {
	io.Writer
}

func newCarriageReturnWriter(dest io.Writer) (*os.File, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	dest = &carriageReturnWriter{dest}

	go io.CopyBuffer(dest, r, make([]byte, 1))
	return w, nil
}

func (c *carriageReturnWriter) Write(p []byte) (n int, err error) {
	newP := bytes.ReplaceAll(p, []byte("\n"), []byte("\n\r"))
	n, err = c.Writer.Write(newP)
	n -= len(newP) - len(p)
	return
}
