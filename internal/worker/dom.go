package worker

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/jsworker"
)

type DOM struct {
	local *Local
	port  *jsworker.Local
}

func ExecDOM(ctx context.Context, localJS *jsworker.Local, command string, args []string, workingDirectory string, env map[string]string) (*DOM, error) {
	err := localJS.PostMessage(makeInitMessage("dom", command, append([]string{command}, args...), workingDirectory, env), nil)
	if err != nil {
		return nil, err
	}
	local, err := NewLocal(ctx, localJS)
	if err != nil {
		return nil, err
	}
	if err := local.Start(); err != nil {
		return nil, err
	}
	return &DOM{
		local: local,
		port:  localJS,
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
