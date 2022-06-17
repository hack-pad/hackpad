package worker

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
)

type DOM struct {
	local *Local
	port  *jsworker.Local
}

func ExecDOM(ctx context.Context, localJS *jsworker.Local, command string, args []string, workingDirectory string, env map[string]string) (*DOM, error) {
	msg, transfers := makeInitMessage(
		"dom",
		command, append([]string{command}, args...),
		workingDirectory,
		env,
		nil,
	)
	err := localJS.PostMessage(msg, transfers)
	if err != nil {
		return nil, err
	}
	log.Print("NewLocal start")
	local, err := NewLocal(ctx, localJS)
	if err != nil {
		return nil, err
	}
	log.Print("local start")
	if err := local.Start(); err != nil {
		return nil, err
	}
	log.Print("local ready")
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
