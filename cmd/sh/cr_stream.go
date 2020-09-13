package main

import (
	"bytes"
	"io"
)

type carriageReturnWriter struct {
	io.Writer
}

func (c *carriageReturnWriter) Write(p []byte) (n int, err error) {
	return c.Writer.Write(bytes.ReplaceAll(p, []byte("\n"), []byte("\r\n")))
}
