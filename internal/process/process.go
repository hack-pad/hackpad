package process

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/internal/global"
	"github.com/johnstarich/go-wasm/internal/interop"
	"github.com/johnstarich/go-wasm/internal/promise"
	"github.com/johnstarich/go-wasm/log"
	"go.uber.org/atomic"
)

const (
	minPID           = 1
	currentPID       = 1
	currentParentPID = 0
)

func Init() {
	err := fs.MkdirAll(interop.WorkingDirectory(), 0750)
	if err != nil {
		panic(err)
	}
	global.Set("context", &Process{})

	globals := js.Global()

	process := globals.Get("process")
	interop.SetFunc(process, "getuid", geteuid)
	interop.SetFunc(process, "geteuid", geteuid)
	interop.SetFunc(process, "getgid", getegid)
	interop.SetFunc(process, "getegid", getegid)
	interop.SetFunc(process, "getgroups", getgroups)
	process.Set("pid", currentPID)
	process.Set("ppid", currentParentPID)
	interop.SetFunc(process, "umask", umask)
	interop.SetFunc(process, "cwd", cwd)
	interop.SetFunc(process, "chdir", chdir)

	globals.Set("child_process", map[string]interface{}{})
	childProcess := globals.Get("child_process")
	interop.SetFunc(childProcess, "spawn", spawn)
	//interop.SetFunc(childProcess, "spawnSync", spawnSync) // TODO is there any way to run spawnSync so we don't hit deadlock?
	interop.SetFunc(childProcess, "wait", wait)
	interop.SetFunc(childProcess, "waitSync", waitSync)
}

var (
	pids    = make(map[PID]*Process)
	lastPID = atomic.NewUint64(minPID)
)

type PID uint64

func (p PID) JSValue() js.Value {
	return js.ValueOf(uint64(p))
}

func (p PID) String() string {
	return fmt.Sprintf("%d", p)
}

type Process struct {
	pid     PID
	command string
	args    []string
	state   string

	err  error
	done chan struct{}
}

func New(command string, args []string) *Process {
	return &Process{
		pid:     PID(lastPID.Inc()),
		command: command,
		args:    args,
		done:    make(chan struct{}),
	}
}

func (p *Process) Start() error {
	return p.startWasm()
}

func (p *Process) Wait() error {
	<-p.done
	return p.err
}

func (p *Process) startWasm() error {
	pids[p.pid] = p
	p.state = "pending"
	log.Printf("Spawning process [%d] %q: %s", p.pid, p.command, strings.Join(p.args, " "))
	buf, err := fs.ReadFile(p.command)
	if err != nil {
		return err
	}
	go p.runWasmBytes(buf)
	return nil
}

func (p *Process) runWasmBytes(wasm []byte) {
	handleErr := func(err error) {
		p.state = "done"
		if err != nil {
			log.Errorf("Failed to start process: %s", err.Error())
			p.err = err
			p.state = "error"
		}
		close(p.done)
	}

	p.state = "compiling wasm"
	goInstance := jsGo.New()
	goInstance.Set("argv", interop.SliceFromStrings(p.args))
	importObject := goInstance.Get("importObject")
	jsBuf := uint8Array.New(len(wasm))
	js.CopyBytesToJS(jsBuf, wasm)
	// TODO add module caching
	instantiatePromise := promise.New(jsWasm.Call("instantiate", jsBuf, importObject))
	module, err := promise.Await(instantiatePromise)
	if err != nil {
		handleErr(err)
		return
	}

	instance := module.Get("instance")
	exports := instance.Get("exports")

	runFn := exports.Get("run")
	resumeFn := exports.Get("resume")
	wrapperExports := map[string]interface{}{
		"run": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			prev := switchContext(p.pid)
			ret := runFn.Invoke(interop.SliceFromJSValues(args)...)
			switchContext(prev)
			return ret
		}),
		"resume": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			prev := switchContext(p.pid)
			ret := resumeFn.Invoke(interop.SliceFromJSValues(args)...)
			switchContext(prev)
			return ret
		}),
	}
	for export, value := range interop.Entries(exports) {
		_, overridden := wrapperExports[export]
		if !overridden {
			wrapperExports[export] = value
		}
	}
	instance = js.ValueOf(map[string]interface{}{ // Instance.exports is read-only, so create a shim
		"exports": wrapperExports,
	})

	p.state = "running"
	runPromise := promise.New(goInstance.Call("run", instance))
	_, err = promise.Await(runPromise)
	handleErr(err)
}

func (p *Process) JSValue() js.Value {
	if p == nil {
		return js.ValueOf(map[string]interface{}{
			"pid": currentPID,
		})
	}
	return js.ValueOf(map[string]interface{}{
		"pid": p.pid,
	})
}

func (p *Process) String() string {
	return fmt.Sprintf("PID=%s, State=%s, Err=%+v", p.pid, p.state, p.err)
}

func switchContext(pid PID) (prev PID) {
	prev = PID(global.Get("context").Get("pid").Int())
	log.Debug("Switching context from PID ", prev, " to ", pid)
	global.Set("context", pids[pid])
	return
}

/*
func environ() map[string]interface{} {
	env := make(map[string]interface{})
	for _, pair := range os.Environ() {
		equalsIndex := strings.IndexRune(pair, '=')
		if equalsIndex == -1 {
			env[pair] = ""
		} else {
			key, val := pair[:equalsIndex], pair[equalsIndex+1:]
			env[key] = val
		}
	}
	return env
}
*/

func Dump() interface{} {
	return fmt.Sprintf("%+v", pids)
}
