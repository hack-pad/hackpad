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
	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/indexeddb/idbblob"
)

type Remote struct {
	pid           common.PID
	port          *jsworker.Remote
	closeCtx      context.Context
	closeExitCode *int
	closeErr      error
}

func NewRemote(local *Local, pid process.PID, command string, argv []string, attr *process.ProcAttr) (*Remote, error) {
	ctx := context.Background()
	closeCtx, cancel := context.WithCancel(ctx)

	var openFiles []openFile
	for _, f := range attr.Files {
		file, err := local.process.Files().RawFID(f.FID)
		if err != nil {
			return nil, err
		}
		info, err := file.Stat()
		if err != nil {
			return nil, err
		}
		openF := openFile{
			filePath:   info.Name(),
			seekOffset: 0, // TODO expose seek offset in file descriptor
		}
		if info.Mode()&hackpadfs.ModeNamedPipe != 0 {
			log.Print("Found pipe, creating MessageChannel...")
			port1, port2, err := jsworker.NewChannel()
			if err != nil {
				return nil, err
			}
			openF.pipe = port1
			log.Print("Connecting port to file...")
			err = connectPortToFile(closeCtx, port2, file)
			if err != nil {
				return nil, err
			}
			log.Print("Connected port to file.")
		}
		openFiles = append(openFiles, openF)
	}
	workerName := fmt.Sprintf("pid-%d", pid)
	port, err := jsworker.NewRemoteWasm(workerName, "/wasm/worker.wasm")
	if err != nil {
		return nil, err
	}

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
			cancel()
			log.Warn("Remote exited with code:", exitCode)
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		log.Print("Worker ", workerName, " awaiting pending_init...")
		err := awaitMessage(ctx, port, "pending_init")
		if err != nil {
			log.Error("Failed awaiting pending_init:", workerName, err)
			return
		}
		log.Print("Worker ", workerName, " waiting to init. Sending init...")
		msg, transfers := makeInitMessage(workerName, command, argv, attr.Dir, attr.Env, openFiles)
		err = port.PostMessage(msg, transfers)
		if err != nil {
			log.Error("Failed sending init to worker: ", workerName, " ", err)
			return
		}
		log.Print("Sent init message to worker ", workerName, ". Awaiting ready...")
		if err := awaitMessage(ctx, remote.port, "ready"); err != nil {
			log.Error("Failed awaiting ready:", workerName, err)
			return
		}
		log.Print("Worker ", workerName, " is ready. Sending start message.")
		err = remote.port.PostMessage(makeStartMessage(), nil)
		if err != nil {
			log.Error("Failed sending start to worker: ", workerName, " ", err)
			return
		}
		log.Print("Sent start message.")
	}()

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

func makeInitMessage(
	workerName,
	command string, argv []string,
	workingDirectory string,
	env map[string]string,
	openFiles []openFile,
) (msg js.Value, transfers []js.Value) {
	var openFileJSValues []interface{}
	var ports []js.Value
	for _, o := range openFiles {
		openFileJSValues = append(openFileJSValues, o)
		if o.pipe != nil {
			ports = append(ports, o.pipe.JSValue())
		}
	}
	return js.ValueOf(map[string]interface{}{
		"init": map[string]interface{}{
			"workerName":       workerName,
			"command":          command,
			"argv":             interop.SliceFromStrings(argv),
			"workingDirectory": workingDirectory,
			"env":              interop.StringMap(env),
			"openFiles":        openFileJSValues,
		},
	}), ports
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

func connectPortToFile(ctx context.Context, port *jsworker.MessagePort, file hackpadfs.File) error {
	return port.Listen(ctx, func(me jsworker.MessageEvent, err error) {
		if err != nil {
			log.Error(err)
			return
		}
		bl, err := idbblob.New(me.Data)
		if err != nil {
			log.Error(err)
			return
		}
		log.Print("Received message! ", string(bl.Bytes()))
		_, err = hackpadfs.WriteFile(file, bl.Bytes())
		if err != nil {
			log.Error(err)
			return
		}
	})
}
