package worker

import (
	"context"
	"io"
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
	localJS         *jsworker.Local
	process         *process.Process
	processStartCtx context.Context
	pids            map[common.PID]*Remote
}

func NewLocal(ctx context.Context, localJS *jsworker.Local) (_ *Local, err error) {
	local := &Local{
		localJS: localJS,
		pids:    make(map[common.PID]*Remote),
	}
	init, err := local.awaitInit(ctx)
	if err != nil {
		return nil, err
	}
	defer common.CatchException(&err)

	global.Set("workerName", init.Get("workerName"))
	log.Debug("Setting process details...")
	local.process, err = process.New(
		kernel.ReservePID(),
		init.Get("command").String(),
		interop.StringsFromJSValue(init.Get("argv")),
		init.Get("workingDirectory").String(),
		parseOpenFiles(init.Get("openFiles")),
		interop.StringMapFromJSObject(init.Get("env")),
	)
	if err != nil {
		return nil, err
	}
	log.Debug("Initializing process")
	jsprocess.Init(local.process, local, local)
	log.Debug("Initializing fs")
	jsfs.Init(local.process)
	return local, nil
}

func (l *Local) awaitInit(ctx context.Context) (js.Value, error) {
	log.Debug("NewLocal 1")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	l.processStartCtx = ctx

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
	err = l.localJS.PostMessage(js.ValueOf("pending_init"), nil)
	if err != nil {
		return js.Value{}, err
	}
	log.Debug("NewLocal 2")
	message := <-initChan
	log.Debug("NewLocal 3")
	return message.init, message.err
}

func (l *Local) Start() (err error) {
	defer common.CatchException(&err)
	startCtx, cancel := context.WithCancel(context.Background())
	err = l.localJS.Listen(startCtx, func(me jsworker.MessageEvent, err error) {
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

		err = l.process.Start()
		if err != nil {
			log.Error(err)
			return
		}
	})
	if err != nil {
		return err
	}

	global.Set("ready", true)
	log.Debug("before ready post")
	err = l.localJS.PostMessage(js.ValueOf("ready"), nil)
	if err != nil {
		return err
	}
	log.Debug("after ready post")
	return nil
}

func (l *Local) Exit(exitCode int) error {
	err := l.localJS.PostMessage(makeExitMessage(exitCode), nil)
	if err != nil {
		return err
	}
	return l.localJS.Close()
}

func (l *Local) Spawn(command string, argv []string, attr *process.ProcAttr) (jsprocess.PIDer, error) {
	pid := kernel.ReservePID()
	log.Debug("Spawning pid ", pid, " for command: ", command, argv)
	remote, err := NewRemote(l, pid, command, argv, attr)
	if err != nil {
		return nil, err
	}
	l.pids[pid] = remote
	return remote, nil
}

func (l *Local) Wait(pid common.PID) (exitCode int, err error) {
	log.Debug("Waiting on pid ", pid)
	if pid == l.process.PID() {
		return l.process.Wait()
	}
	remote, ok := l.pids[pid]
	if !ok {
		return 0, errors.Errorf("Unknown child process: %d", pid)
	}
	return remote.Wait()
}

func (l *Local) Started() <-chan struct{} {
	return l.processStartCtx.Done()
}

func (l *Local) PID() common.PID {
	return l.process.PID()
}

func makeExitMessage(exitCode int) js.Value {
	return js.ValueOf(map[string]interface{}{
		"exitCode": exitCode,
	})
}

func parseOpenFiles(v js.Value) []common.OpenFileAttr {
	openFileJSValues := interop.SliceFromJSValue(v)
	var openFiles []common.OpenFileAttr
	for _, o := range openFileJSValues {
		openFile := readOpenFile(o)
		var pipe io.ReadWriteCloser
		if openFile.pipe != nil {
			var err error
			pipe, err = portToReadWriteCloser(openFile.pipe)
			if err != nil {
				panic(err)
			}
		}
		openFiles = append(openFiles, common.OpenFileAttr{
			FilePath:   openFile.filePath,
			SeekOffset: openFile.seekOffset,
			RawDevice:  pipe,
		})
	}
	return openFiles
}
