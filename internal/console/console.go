package console

import (
	"io"
)

type Console interface {
	Stdout() io.Writer
	Stderr() io.Writer
	Note() io.Writer
}
