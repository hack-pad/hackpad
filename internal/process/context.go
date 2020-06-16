package process

import (
	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/log"
)

var (
	currentPID PID

	switchedContextListener func(newPID, parentPID PID)
)

func Init(switchedContext func(PID, PID)) {
	pids[minPID] = &process{
		pid:             minPID,
		fileDescriptors: fs.NewFileDescriptors(),
	}
	switchedContextListener = switchedContext
	switchContext(minPID)
}

func switchContext(pid PID) (prev PID) {
	prev = currentPID
	log.Debug("Switching context from PID ", prev, " to ", pid)
	newProcess := pids[pid]
	currentPID = pid
	switchedContextListener(pid, newProcess.parentPID)
	return
}

func Current() Process {
	process, _ := Get(currentPID)
	return process
}

func Get(pid PID) (process Process, ok bool) {
	p, ok := pids[pid]
	if ok {
		pCopy := *p
		return &pCopy, ok
	}
	return
}
