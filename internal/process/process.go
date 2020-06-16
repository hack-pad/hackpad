package process

import (
	"fmt"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"go.uber.org/atomic"
)

const (
	minPID = 1
)

var (
	jsGo       = js.Global().Get("Go")
	jsWasm     = js.Global().Get("WebAssembly")
	uint8Array = js.Global().Get("Uint8Array")
)

var (
	pids    = make(map[PID]*process)
	lastPID = atomic.NewUint64(minPID)
)

type Process interface {
	PID() PID
	ParentPID() PID

	Start() error
	Wait() error
	Files() *fs.FileDescriptors
	WorkingDirectory() string
	SetWorkingDirectory(wd string) error
}

type process struct {
	pid, parentPID  PID
	command         string
	args            []string
	state           string
	attr            *ProcAttr
	done            chan struct{}
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
	files, setFilesWD, err := fs.NewFileDescriptors(wd, current.Files(), attr.Files)
	return &process{
		pid:             newPID,
		command:         command,
		args:            args,
		state:           "pending",
		attr:            attr,
		done:            make(chan struct{}),
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
	return p.startWasm()
}

func (p *process) Wait() error {
	<-p.done
	return p.err
}

func (p *process) WorkingDirectory() string {
	return p.Files().WorkingDirectory()
}

func (p *process) SetWorkingDirectory(wd string) error {
	return p.setFilesWD(wd)
}

func (p *process) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"pid":  p.pid,
		"ppid": p.parentPID,
	})
}

func (p *process) String() string {
	return fmt.Sprintf("PID=%s, State=%s, WD=%s, Attr=%+v, Err=%+v", p.pid, p.state, p.WorkingDirectory(), p.attr, p.err)
}

func Dump() interface{} {
	return fmt.Sprintf("%v", pids)
}
