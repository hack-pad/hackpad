package process

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpadfs/keyvalue/blob"
	"github.com/pkg/errors"
	"go.uber.org/atomic"
)

const (
	minPID = 1
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
	pids    = make(map[PID]*process)
	lastPID = atomic.NewUint64(minPID)
)

type Process interface {
	PID() PID
	ParentPID() PID

	Start() error
	Wait() (exitCode int, err error)
	Files() *fs.FileDescriptors
	WorkingDirectory() string
	SetWorkingDirectory(wd string) error
}

type process struct {
	pid, parentPID  PID
	command         string
	args            []string
	state           processState
	attr            *ProcAttr
	ctx             context.Context
	ctxDone         context.CancelFunc
	exitCode        int
	err             error
	fileDescriptors *fs.FileDescriptors
	setFilesWD      func(wd string) error
}

func New(command string, args []string, attr *ProcAttr) (Process, error) {
	return newWithCurrent(Current(), PID(lastPID.Inc()), command, args, attr)
}

func newWithCurrent(current Process, newPID PID, command string, args []string, attr *ProcAttr) (*process, error) {
	wd := current.WorkingDirectory()
	if attr.Dir != "" {
		wd = attr.Dir
	}
	files, setFilesWD, err := fs.NewFileDescriptors(newPID, wd, current.Files(), attr.Files)
	ctx, cancel := context.WithCancel(context.Background())
	return &process{
		pid:             newPID,
		command:         command,
		args:            args,
		state:           statePending,
		attr:            attr,
		ctx:             ctx,
		ctxDone:         cancel,
		err:             err,
		fileDescriptors: files,
		setFilesWD:      setFilesWD,
	}, err
}

func (p *process) PID() PID {
	return p.pid
}

func (p *process) ParentPID() PID {
	return p.parentPID
}

func (p *process) Files() *fs.FileDescriptors {
	return p.fileDescriptors
}

func (p *process) Start() error {
	err := p.start()
	if p.err == nil {
		p.err = err
	}
	return p.err
}

func (p *process) start() error {
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

func (p *process) prepExecutable() (command string, err error) {
	fs := p.Files()
	command, err = lookPath(fs.Stat, os.Getenv("PATH"), p.command)
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

func (p *process) Done() {
	log.Debug("PID ", p.pid, " is done.\n", p.fileDescriptors)
	p.fileDescriptors.CloseAll()
	p.ctxDone()
}

func (p *process) handleErr(err error) {
	p.state = stateDone
	if err != nil {
		log.Errorf("Failed to start process: %s", err.Error())
		p.err = err
		p.state = stateError
	}
	p.Done()
}

func (p *process) Wait() (exitCode int, err error) {
	<-p.ctx.Done()
	return p.exitCode, p.err
}

func (p *process) WorkingDirectory() string {
	return p.Files().WorkingDirectory()
}

func (p *process) SetWorkingDirectory(wd string) error {
	return p.setFilesWD(wd)
}

func (p *process) String() string {
	return fmt.Sprintf("PID=%s, Command=%v, State=%s, WD=%s, Attr=%+v, Err=%+v, Files:\n%v", p.pid, p.args, p.state, p.WorkingDirectory(), p.attr, p.err, p.fileDescriptors)
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
