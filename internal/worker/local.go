package worker

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/global"
	"github.com/hack-pad/hackpad/internal/interop"
	jsfs "github.com/hack-pad/hackpad/internal/js/fs"
	jsprocess "github.com/hack-pad/hackpad/internal/js/process"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/kernel"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/process"
	"github.com/pkg/errors"
)

type Local struct {
	localJS *jsworker.Local
	process *process.Process
	pids    map[common.PID]*Remote
}

func NewLocal(localJS *jsworker.Local) (_ *Local, err error) {
	local := &Local{
		localJS: localJS,
		pids:    make(map[common.PID]*Remote),
	}

	init, err := local.awaitInit(context.Background())
	if err != nil {
		return nil, err
	}

	defer common.CatchException(&err)
	local.process, err = process.New(
		kernel.ReservePID(),
		init.Get("command").String(),
		interop.StringsFromJSValue(init.Get("argv")),
		init.Get("workingDirectory").String(),
		nil, // TODO open files
		interop.StringMapFromJSObject(init.Get("env")),
	)
	if err != nil {
		return nil, err
	}
	jsprocess.Init(local.process, local, local)
	jsfs.Init(local.process)
	global.Set("ready", true)
	log.Debug("before ready post")
	localJS.PostMessage(js.ValueOf("ready"), nil)
	log.Debug("after ready post")

	local.listenStart()

	return local, nil
}

func (l *Local) awaitInit(ctx context.Context) (js.Value, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type initMessage struct {
		err  error
		init js.Value
	}
	initChan := make(chan initMessage, 1)
	err := l.localJS.Listen(ctx, func(me jsworker.MessageEvent, err error) {
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
	if err != nil {
		return js.Value{}, err
	}
	message := <-initChan
	return message.init, message.err
}

func (l *Local) listenStart() error {
	startCtx, cancel := context.WithCancel(context.Background())
	return l.localJS.Listen(startCtx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			log.Error(err)
			cancel()
			return
		}
		defer common.CatchExceptionHandler(func(err error) {
			log.Error(err)
			cancel()
		})
		if me.Data.Type() != js.TypeObject {
			return
		}
		entries := interop.Entries(me.Data)
		_, ok := entries["start"]
		if !ok {
			return
		}
		cancel()

		log.Print("Starting process: ", l.process.PID)
		err = l.process.Start()
		if err != nil {
			log.Error(err)
			return
		}
	})
}

func (l *Local) Spawn(command string, argv []string, attr *process.ProcAttr) (jsprocess.PIDer, error) {
	pid := kernel.ReservePID()
	log.Print("Spawning pid: ", pid, " for command: ", command, argv)
	return NewRemote(l, pid, command, argv, attr)
}

func (l *Local) Wait(pid common.PID) (exitCode int, err error) {
	log.Print("Waiting on pid: ", pid)
	remote, ok := l.pids[pid]
	if !ok {
		return 0, errors.Errorf("Unknown child process: %d", pid)
	}
	return remote.Wait()
}
