package worker

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/process"
)

type Remote struct {
	pid           common.PID
	port          *jsworker.Remote
	closeCtx      context.Context
	closeExitCode *int
	closeErr      error
}

type openFile struct {
	filePath   string
	seekOffset uint
}

func NewRemote(local *Local, pid process.PID, command string, argv []string, attr *process.ProcAttr) (*Remote, error) {
	var openFiles []openFile
	for _, f := range attr.Files {
		info, err := local.process.Files().Fstat(f.FID)
		if err != nil {
			return nil, err
		}
		openFiles = append(openFiles, openFile{
			filePath:   info.Name(),
			seekOffset: 0, // TODO expose seek offset in file descriptor
		})
	}
	// TODO inherit file descriptors
	port, err := jsworker.NewRemoteWasm(fmt.Sprintf("pid-%d", pid), "/wasm/worker.wasm")
	if err != nil {
		return nil, err
	}
	err = port.PostMessage(makeInitMessage(command, argv, attr.Dir, attr.Env), nil)
	if err != nil {
		return nil, err
	}

	closeCtx, cancel := context.WithCancel(context.Background())
	remote := &Remote{
		pid:      pid,
		port:     port,
		closeCtx: closeCtx,
	}

	err = port.Listen(closeCtx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			remote.closeErr = err
			cancel()
			return
		}
		if me.Data.Type() != js.TypeObject {
			return
		}
		data := interop.Entries(me.Data)
		if jsExitCode, ok := data["exitCode"]; ok && jsExitCode.Type() == js.TypeNumber {
			exitCode := jsExitCode.Int()
			remote.closeExitCode = &exitCode
		}
		cancel()
	})
	if err != nil {
		return nil, err
	}

	err = remote.port.PostMessage(makeStartMessage(), nil)
	if err != nil {
		return nil, err
	}

	return remote, nil
}

func (r *Remote) PID() common.PID {
	return r.pid
}

func makeInitMessage(command string, argv []string, workingDirectory string, env map[string]string) js.Value {
	return js.ValueOf(map[string]interface{}{
		"init": map[string]interface{}{
			"command":          command,
			"argv":             interop.SliceFromStrings(argv),
			"workingDirectory": workingDirectory,
			"env":              interop.StringMap(env),
		},
	})
}

func (r *Remote) Wait() (exitCode int, err error) {
	<-r.closeCtx.Done()
	if r.closeExitCode == nil {
		switch {
		case r.closeErr != nil:
			return 0, r.closeErr
		default:
			return 0, r.closeCtx.Err()
		}
	}
	return *r.closeExitCode, r.closeErr
}
