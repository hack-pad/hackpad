package process

import (
	"github.com/johnstarich/go-wasm/internal/fs"
	"github.com/johnstarich/go-wasm/log"
)

const initialDirectory = "/home/me"

var (
	currentPID PID

	switchedContextListener func(newPID, parentPID PID)
)

func Init(switchedContext func(PID, PID)) {
	fileDescriptors, err := fs.NewStdFileDescriptors(initialDirectory)
	if err != nil {
		panic(err)
	}
	pids[minPID], err = newWithCurrent(
		&process{fileDescriptors: fileDescriptors},
		minPID,
		"",
		nil,
		&ProcAttr{},
	)
	if err != nil {
		panic(err)
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
