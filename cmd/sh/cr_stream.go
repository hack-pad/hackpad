package main

import (
	"bytes"
	"io"
)

type carriageReturnWriter struct {
	io.Writer
}

func (c *carriageReturnWriter) Write(p []byte) (n int, err error) {
	newP := bytes.ReplaceAll(p, []byte("\n"), []byte("\r\n"))
	n, err = c.Writer.Write(newP)
	n -= len(newP) - len(p)
	return
}
