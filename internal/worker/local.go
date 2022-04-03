package worker

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	jsfs "github.com/hack-pad/hackpad/internal/js/fs"
	jsprocess "github.com/hack-pad/hackpad/internal/js/process"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/kernel"
	"github.com/hack-pad/hackpad/internal/process"
)

type Local struct {
	process *process.Process
}

func NewLocal(localJS *jsworker.Local) (*Local, error) {
	ctx, cancel := context.WithCancel(context.Background())
	type initMessage struct {
		err  error
		init js.Value
	}
	initChan := make(chan initMessage, 1)
	localJS.Listen(ctx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			initChan <- initMessage{err: err}
			return
		}
		if !me.Data.Truthy() || me.Data.Type() != js.TypeObject {
			return
		}
		initData := me.Data.Get("init")
		if !initData.Truthy() {
			return
		}
		initChan <- initMessage{init: initData}
	})
	message := <-initChan
	cancel()
	if message.err != nil {
		return nil, message.err
	}

	process, err := process.New(
		kernel.ReservePID(),
		message.init.Get("command").String(),
		interop.StringsFromJSValue(message.init.Get("args")),
		message.init.Get("workingDirectory").String(),
		nil, // TODO open files
		interop.StringMapFromJSObject(message.init.Get("env")),
	)
	if err != nil {
		return nil, err
	}
	local := &Local{
		process: process,
	}
	jsprocess.Init(local.process)
	jsfs.Init(local.process)
	global.Set("ready", true)
	localJS.PostMessage(js.ValueOf("ready"), nil)
	return local, nil
}

func (l *Local) Fork(command string, args []string, attr *process.ProcAttr) (*Remote, error) {
	remote, err := NewRemote(l, kernel.ReservePID(), command, args, attr)
	// TODO worker init
	return remote, err
}
