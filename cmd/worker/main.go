package main

import (
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/worker"
)

func main() {
	log.Warn("booting worker")
	jsLocal := jsworker.GetLocal()
	local, err := worker.NewLocal(jsLocal)
	if err != nil {
		panic(err)
	}
	log.Warn("worker started:", local)
	select {}
}
