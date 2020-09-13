package process

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/interop"
	"go.uber.org/atomic"
)

const (
	minPID    = 1
	stdinFID  = 0 // TODO should these be in 'fs'?
	stdoutFID = 1
	stderrFID = 2
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
	jsGo   = js.Global().Get("Go")
	jsWasm = js.Global().Get("WebAssembly")
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
	err := p.startWasm()
	if p.err == nil {
		p.err = err
	}
	return p.err
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

func (p *process) JSValue() js.Value {
	stdio := p.fileDescriptors.RawFIDs()
	stdin := newWritableStream(p.ctx, stdio[stdinFID])
	stdout := newReadableStream(p.ctx, stdio[stdoutFID], js.Null())
	stderr := newReadableStream(p.ctx, stdio[stderrFID], js.Null())
	return js.ValueOf(map[string]interface{}{
		"pid":    p.pid,
		"ppid":   p.parentPID,
		"error":  interop.WrapAsJSError(p.err, "spawn"),
		"stdio":  []interface{}{stdin, stdout, stderr},
		"stdin":  stdin,
		"stdout": stdout,
		"stderr": stderr,
	})
}

func (p *process) String() string {
	return fmt.Sprintf("PID=%s, Command=%v, State=%s, WD=%s, Attr=%+v, Err=%+v, Files:\n%v", p.pid, p.args, p.state, p.WorkingDirectory(), p.attr, p.err, p.fileDescriptors)
}

func (p *process) StartCPUProfile() error {
	return interop.StartCPUProfile(p.ctx)
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
