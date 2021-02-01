// +build js

package queue

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/indexeddb"
)

type Queue struct {
	ops chan opRunner
}

type Op = func(*indexeddb.Transaction) js.Value

func New(size int) *Queue {
	return &Queue{
		ops: make(chan opRunner, size),
	}
}

type opRunner struct {
	mode       indexeddb.TransactionMode
	storeNames []string
	op         Op
	result     chan<- js.Value
	err        chan<- error
}

func (q *Queue) Push(mode indexeddb.TransactionMode, storeNames []string, op Op) (<-chan js.Value, <-chan error) {
	result := make(chan js.Value, 1)
	err := make(chan error, 1)
	q.ops <- opRunner{
		mode:       mode,
		storeNames: storeNames,
		op:         op,
		result:     result,
		err:        err,
	}
	return result, err
}

func (q *Queue) Do(db *indexeddb.DB) ([]js.Value, error) {
	runners := q.readOps()
	mode := indexeddb.TransactionReadOnly
	storeNameSet := make(map[string]bool)
	for _, runner := range runners {
		if runner.mode == indexeddb.TransactionReadWrite {
			mode = runner.mode
		}
		for _, storeName := range runner.storeNames {
			storeNameSet[storeName] = true
		}
	}
	storeNames := make([]string, 0, len(storeNameSet))
	for name := range storeNameSet {
		storeNames = append(storeNames, name)
	}

	var results []js.Value
	requests, err := runOps(db, mode, storeNames, runners)
	if err == nil {
		results = make([]js.Value, len(requests))
		for i := range runners {
			result := requests[i].Get("result")
			runners[i].result <- result
			results[i] = result
		}
	}
	for i := range runners {
		runners[i].err <- err
		close(runners[i].err)
		close(runners[i].result) // always close result chans, even on error
	}
	return results, err
}

func (q *Queue) readOps() []opRunner {
	var runners []opRunner
	for {
		select {
		case runner := <-q.ops:
			runners = append(runners, runner)
		default:
			return runners
		}
	}
}

func runOps(db *indexeddb.DB, mode indexeddb.TransactionMode, storeNames []string, runners []opRunner) (requests []js.Value, err error) {
	txn, err := db.Transaction(mode, storeNames...)
	if err != nil {
		return nil, err
	}

	for _, runner := range runners {
		requests = append(requests, runner.op(txn))
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}
	err = txn.Await()
	return requests, err
}
