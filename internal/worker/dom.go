package worker

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/jsworker"
)

type DOM struct {
	local *Local
	port  *jsworker.Local
}

func ExecDOM(localJS *jsworker.Local, command string, args []string, workingDirectory string, env map[string]string) (*DOM, error) {
	localJS.PostMessage(makeInitMessage(command, args, workingDirectory, env), nil)
	local, err := NewLocal(localJS)
	if err != nil {
		return nil, err
	}
	return &DOM{
		local: local,
	}, nil
}

func (d *DOM) Start() error {
	return d.port.PostMessage(makeStartMessage(), nil)
}

func makeStartMessage() js.Value {
	return js.ValueOf(map[string]interface{}{
		"start": true,
	})
}
