// +build js

package queue

import (
	"context"
	"sync"
	"syscall/js"
	"time"

	"github.com/johnstarich/go-wasm/internal/indexeddb"
)

type Queue struct {
	ops        chan opRunner
	startOnce  sync.Once
	pushNotifs chan struct{}
}

func New(size int) *Queue {
	return &Queue{
		ops: make(chan opRunner, size),
	}
}

type opRunner struct {
	mode       indexeddb.TransactionMode
	storeNames []string
	op         indexeddb.Op
	result     chan<- js.Value
	err        chan<- error
}

func (q *Queue) Push(mode indexeddb.TransactionMode, storeNames []string, op indexeddb.Op) (<-chan js.Value, <-chan error) {
	result := make(chan js.Value, 1)
	err := make(chan error, 1)
	q.ops <- opRunner{
		mode:       mode,
		storeNames: storeNames,
		op:         op,
		result:     result,
		err:        err,
	}
	if q.pushNotifs != nil {
		q.pushNotifs <- struct{}{}
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
			result, resultErr := requests[i].Result()
			if resultErr != nil && err == nil {
				err = resultErr
				break
			}
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

func runOps(db *indexeddb.DB, mode indexeddb.TransactionMode, storeNames []string, runners []opRunner) (requests []*indexeddb.Request, err error) {
	if len(runners) == 0 {
		return nil, nil
	}
	txn, err := db.Transaction(mode, storeNames...)
	if err != nil {
		return nil, err
	}

	for _, runner := range runners {
		req, err := runner.op(txn)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	err = txn.Commit()
	if err != nil {
		return nil, err
	}
	err = txn.Await()
	return requests, err
}

func (q *Queue) StartAsync(ctx context.Context, interval time.Duration, db *indexeddb.DB) {
	q.startOnce.Do(func() {
		q.pushNotifs = make(chan struct{}, cap(q.ops))
		go q.runDoLoop(ctx, interval, db)
	})
}

func (q *Queue) runDoLoop(ctx context.Context, interval time.Duration, db *indexeddb.DB) {
	maxSize := float64(cap(q.ops))
	timer := time.NewTimer(interval)
	prevInterval := interval

	const (
		minWaitPercent = 1   // must wait at least this long %-wise. needed to refill the queue
		maxWaitPercent = 100 // max wait, for responsiveness
	)
	minInterval := time.Duration(float64(interval) * minWaitPercent)
	maxInterval := time.Duration(float64(interval) * maxWaitPercent)
	for {
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-q.pushNotifs:
			if prevInterval > interval {
				if !timer.Stop() {
					<-timer.C
					timer.Reset(0)
				} else {
					prevInterval = minInterval
					timer.Reset(prevInterval)
				}
			}
		case <-timer.C:
			results, _ := q.Do(db) // errors handled by returned channels from q.Push()

			/*
				Reduce busy-wait with wait time multiplier:

				use fill % to calculate next wait interval

				low %:    0% filled -> 1.10x prev wait
				          1% filled -> 0.50x prev wait
				         25% filled -> 0.25x prev wait
				         50% filled -> 0.10x prev wait
				         75% filled -> 0.10x prev wait
				high %: 100% filled -> 0.10x prev wait
			*/

			fill := float64(len(results)) / maxSize
			var waitMultiplier float64
			switch {
			case fill >= 0.50:
				waitMultiplier = 0.10
			case fill >= 0.25:
				waitMultiplier = 0.25
			case fill > 0.00:
				waitMultiplier = 0.50
			default:
				waitMultiplier = 1.10
			}
			prevInterval = time.Duration(float64(prevInterval) * waitMultiplier)

			if prevInterval < minInterval {
				prevInterval = minInterval
			} else if prevInterval > maxInterval {
				prevInterval = maxInterval
			}
			timer.Reset(prevInterval)
		}
	}
}
