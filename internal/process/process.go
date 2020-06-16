package process

import (
	"fmt"
	"syscall/js"

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
}

type process struct {
	pid, parentPID PID
	command        string
	args           []string
	state          string

	err  error
	done chan struct{}
}

func New(command string, args []string) Process {
	return &process{
		pid:     PID(lastPID.Inc()),
		command: command,
		args:    args,
		state:   "pending",

		done: make(chan struct{}),
	}
}

func (p *process) PID() PID {
	return p.pid
}

func (p *process) ParentPID() PID {
	return p.parentPID
}

func (p *process) Start() error {
	return p.startWasm()
}

func (p *process) Wait() error {
	<-p.done
	return p.err
}

func (p *process) JSValue() js.Value {
	return js.ValueOf(map[string]interface{}{
		"pid":  p.pid,
		"ppid": p.parentPID,
	})
}

func (p *process) String() string {
	return fmt.Sprintf("PID=%s, State=%s, Err=%+v", p.pid, p.state, p.err)
}

func Dump() interface{} {
	return fmt.Sprintf("%v", pids)
}
