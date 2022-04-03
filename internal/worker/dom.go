package worker

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
)

type DOM struct {
	local *Local
	port  *jsworker.Local
}

func ExecDOM(localJS *jsworker.Local, command string, args []string, workingDirectory string, env map[string]string) (*DOM, error) {
	localJS.PostMessage(js.ValueOf(map[string]interface{}{
		"init": map[string]interface{}{
			"command":          command,
			"args":             interop.SliceFromStrings(args),
			"workingDirectory": workingDirectory,
			"env":              interop.StringMap(env),
		},
	}), nil)
	local, err := NewLocal(localJS)
	if err != nil {
		return nil, err
	}
	return &DOM{
		local: local,
	}, nil
}
