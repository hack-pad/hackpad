package main

import (
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
	log.Warn("booting worker")
	jsLocal := jsworker.GetLocal()
	local, err := worker.NewLocal(jsLocal)
	if err != nil {
		panic(err)
	}
	log.Warn("worker started:", local)
	select {}
}
