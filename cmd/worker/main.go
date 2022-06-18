package main

import (
	"context"
	"os"
	"runtime/debug"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpad/internal/common"
	"github.com/hack-pad/hackpad/internal/fs"
	"github.com/hack-pad/hackpad/internal/jsworker"
	"github.com/hack-pad/hackpad/internal/log"
	"github.com/hack-pad/hackpad/internal/worker"
	"github.com/hack-pad/hackpadfs/indexeddb"
)

func main() {
	defer common.CatchExceptionHandler(func(err error) {
		log.Errorf("Worker panicked: %+v", err)
		log.Error(string(debug.Stack()))
		os.Exit(1)
	})

	log.SetLevel(log.LevelDebug)
	bootCtx := context.Background()
	log.Debug("booting worker")
	local, err := worker.NewLocal(bootCtx, jsworker.GetLocal())
	if err != nil {
		panic(err)
	}
	log.Debug("worker inited")
	if err := setUpFS(); err != nil {
		panic(err)
	}
	log.Debug("fs is setup")
	if err := local.Start(); err != nil {
		panic(err)
	}
	log.Debug("worker starting...")
	<-local.Started()
	pid := local.PID()
	log.Debug("worker process started PID ", pid)
	exitCode, err := local.Wait(pid)
	if err != nil {
		log.Error("Failed to wait for PID ", pid, ":", err)
		exitCode = 1
	}
	log.Warn("worker stopped for PID ", pid, "; exit code = ", exitCode)
	local.Exit(exitCode)
	os.Exit(exitCode)
}

func setUpFS() error {
	const dirPerm = 0700
	if err := os.MkdirAll("/bin", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/bin", idb.DurabilityRelaxed); err != nil {
		return err
	}
	if err := os.MkdirAll("/home/me", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/home/me", idb.DurabilityDefault); err != nil {
		return err
	}
	if err := os.MkdirAll("/home/me/.cache", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/home/me/.cache", idb.DurabilityRelaxed); err != nil {
		return err
	}
	if err := os.MkdirAll("/usr/local/go", dirPerm); err != nil {
		return err
	}
	if err := overlayIndexedDB("/usr/local/go", idb.DurabilityRelaxed); err != nil {
		return err
	}
	return nil
}

func overlayIndexedDB(mountPath string, durability idb.TransactionDurability) error {
	idbFS, err := indexeddb.NewFS(context.Background(), mountPath, indexeddb.Options{
		TransactionDurability: durability,
	})
	if err != nil {
		return err
	}
	return fs.Overlay(mountPath, idbFS)
}
