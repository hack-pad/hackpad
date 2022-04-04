package worker

import (
	"context"
	"fmt"
	"syscall/js"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/interop"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
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
	ctx := context.Background()

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
	workerName := fmt.Sprintf("pid-%d", pid)
	port, err := jsworker.NewRemoteWasm(workerName, "/wasm/worker.wasm")
	if err != nil {
		return nil, err
	}
	err = awaitMessage(ctx, port, "pending_init")
	if err != nil {
		return nil, err
	}
	log.Warn("Sending init to worker ", workerName)
	err = port.PostMessage(makeInitMessage(command, argv, attr.Dir, attr.Env), nil)
	if err != nil {
		return nil, err
	}
	log.Warn("init sent to worker ", workerName)

	closeCtx, cancel := context.WithCancel(ctx)
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

func awaitMessage(ctx context.Context, port *jsworker.Remote, messageStr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error, 1)
	err := port.Listen(ctx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			result <- err
			return
		}
		if me.Data.Type() == js.TypeString && me.Data.String() == messageStr {
			result <- nil
		}
	})
	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-result:
		return err
	}
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
