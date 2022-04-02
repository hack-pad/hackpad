package worker

import (
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/kernel"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/process"
)

type Local struct {
	process *process.Process
}

func New(localJS *jsworker.Local) (*Local, error) {
	ctx, cancel := context.WithCancel(context.Background())
	localChan := make(chan *Local, 1)
	localJS.Listen(ctx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			log.Error(err)
			return
		}
		initData := me.Data.Get("init")
		if !initData.Truthy() {
			return
		}
		process := process.New()
		localChan <- &Local{
			initData.
		}
	})
	return nil, nil
}

func (l *Local) Fork(command string, args []string, attr *process.ProcAttr) (*Remote, error) {
	remote, err := NewRemote(l, kernel.ReservePID(), command, args, attr)
	// TODO worker init
	return remote, err
}
