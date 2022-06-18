package process

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
	"github.com/pkg/errors"
)

type PID = common.PID

type processState string

const (
	statePending   processState = "pending"
	stateCompiling processState = "compiling wasm"
	stateRunning   processState = "running"
	stateDone      processState = "done"
	stateError     processState = "error"
)

var (
	pids = make(map[PID]*Process)
)

type Process struct {
	pid, parentPID  PID
	command         string
	args            []string
	state           processState
	env             map[string]string
	ctx             context.Context
	ctxDone         context.CancelFunc
	exitCode        int
	err             error
	fileDescriptors *fs.FileDescriptors
	setFilesWD      func(wd string) error
}

func New(newPID PID, command string, args []string, workingDirectory string, openFiles []common.OpenFileAttr, env map[string]string) (*Process, error) {
	files, setFilesWD, err := fs.NewFileDescriptors(newPID, workingDirectory, openFiles)
	ctx, cancel := context.WithCancel(context.Background())
	return &Process{
		pid:             newPID,
		command:         command,
		args:            args,
		state:           statePending,
		env:             env,
		ctx:             ctx,
		ctxDone:         cancel,
		err:             err,
		fileDescriptors: files,
		setFilesWD:      setFilesWD,
	}, err
}

func (p *Process) PID() PID {
	return p.pid
}

func (p *Process) ParentPID() PID {
	return p.parentPID
}

func (p *Process) Files() *fs.FileDescriptors {
	return p.fileDescriptors
}

func (p *Process) Start() error {
	err := p.start()
	if p.err == nil {
		p.err = err
	}
	return p.err
}

func (p *Process) start() error {
	pids[p.pid] = p
	log.Debugf("Spawning process: %v", p)
	go func() {
		command, err := p.prepExecutable()
		if err != nil {
			p.handleErr(err)
			return
		}
		p.run(command)
	}()
	return nil
}

func (p *Process) prepExecutable() (command string, err error) {
	fs := p.Files()
	command, err = lookPath(fs.Stat, p.env["PATH"], p.command)
	if err != nil {
		return "", err
	}
	fid, err := fs.Open(command, 0, 0)
	if err != nil {
		return "", err
	}
	defer fs.Close(fid)
	buf := blob.NewBytesLength(4)
	_, err = fs.Read(fid, buf, 0, buf.Len(), nil)
	if err != nil {
		return "", err
	}
	magicNumber := string(buf.Bytes())
	if magicNumber != "\x00asm" {
		return "", errors.Errorf("Format error. Expected Wasm file header but found: %q", magicNumber)
	}
	return command, nil
}

func (p *Process) Done() {
	log.Debug("PID ", p.pid, " is done.\n", p.fileDescriptors)
	p.fileDescriptors.CloseAll()
	p.ctxDone()
}

func (p *Process) handleErr(err error) {
	p.state = stateDone
	if err != nil {
		log.Errorf("Failed to start process: %s", err.Error())
		p.err = err
		p.state = stateError
	}
	p.Done()
}

func (p *Process) Wait() (exitCode int, err error) {
	<-p.ctx.Done()
	return p.exitCode, p.err
}

func (p *Process) WorkingDirectory() string {
	return p.Files().WorkingDirectory()
}

func (p *Process) SetWorkingDirectory(wd string) error {
	return p.setFilesWD(wd)
}

func (p *Process) String() string {
	return fmt.Sprintf("PID=%s, Command=%v, State=%s, WD=%s, Attr=%+v, Err=%+v, Files:\n%v", p.pid, p.args, p.state, p.WorkingDirectory(), p.env, p.err, p.fileDescriptors)
}

func Dump() interface{} {
	var s strings.Builder
	var pidSlice []PID
	for pid := range pids {
		pidSlice = append(pidSlice, pid)
	}
	sort.Slice(pidSlice, func(a, b int) bool {
		return pidSlice[a] < pidSlice[b]
	})
	for _, pid := range pidSlice {
		s.WriteString(pids[pid].String() + "\n")
	}
	return s.String()
}

func (p *Process) Env() map[string]string {
	envCopy := make(map[string]string, len(p.env))
	for k, v := range p.env {
		envCopy[k] = v
	}
	return envCopy
}
