package process

import (
	"github.com/hack-pad/hackpad/internal/fs"
)

// ProcAttr is functionally identical to os.ProcAttr.
// Env is structured as a map (instead of key=value pairs), and files is purely a list of nil-able file descriptor IDs. nil FIDs are to be effectively closed to the new process.
type ProcAttr struct {
	Dir   string
	Env   map[string]string
	Files []fs.Attr
}
