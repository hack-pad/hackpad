package main

import (
	"context"
	"os"

	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/worker"
)

func main() {
	defer common.CatchExceptionHandler(func(err error) {
		log.Error("Worker panicked:", err)
		os.Exit(1)
	})

	log.SetLevel(log.LevelDebug)
	bootCtx := context.Background()
	//bootCtx, bootCancel := context.WithTimeout(context.Background(), 30*time.Second)
	//defer bootCancel()
	log.Print("booting worker")
	local, err := worker.NewLocal(bootCtx, jsworker.GetLocal())
	if err != nil {
		panic(err)
	}
	log.Print("worker started")
	<-local.Started()
	pid := local.PID()
	log.Print("worker process started PID ", pid)
	exitCode, err := local.Wait(pid)
	if err != nil {
		log.Error("Failed to wait for PID ", pid, ":", err)
		exitCode = 1
	}
	log.Warn("worker stopped for PID ", pid, "; exit code = ", exitCode)
	local.Exit(exitCode)
	os.Exit(exitCode)
}
