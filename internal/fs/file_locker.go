package fs

import (
	"context"
	"errors"
	"syscall/js"
	"time"

	"github.com/hack-pad/go-indexeddb/idb"
)

type fileLocker struct {
	db *idb.Database
}

const (
	locksObjectStore = "locks"

	sharedCountField = "sharedCount"
)

func newFileLocker(ctx context.Context) (*fileLocker, error) {
	openRequest, err := idb.Global().Open(ctx, "file-locks", 1, func(db *idb.Database, oldVersion, newVersion uint) error {
		_, err := db.CreateObjectStore(locksObjectStore, idb.ObjectStoreOptions{})
		return err
	})
	if err != nil {
		return nil, err
	}
	db, err := openRequest.Await(ctx)
	if err != nil {
		return nil, err
	}
	return &fileLocker{
		db: db,
	}, nil
}

func (f *fileLocker) Lock(ctx context.Context, filePath string, shared bool) error {
	locked, err := f.tryLock(ctx, filePath, shared)
	if locked || err != nil {
		return err
	}

	// lock is either exclusive or does not match the current lock type. must wait its turn.
	const pollInterval = 10 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			drainTicker(ticker)
			locked, err := f.tryLock(ctx, filePath, shared)
			if locked || err != nil {
				return err
			}
		}
	}
}

func (f *fileLocker) tryLock(ctx context.Context, filePath string, shared bool) (locked bool, err error) {
	txn, err := f.db.Transaction(idb.TransactionReadWrite, locksObjectStore)
	if err != nil {
		return false, err
	}
	locks, err := txn.ObjectStore(locksObjectStore)
	if err != nil {
		return false, err
	}
	jsKey := js.ValueOf(filePath)
	req, err := locks.Get(jsKey)
	if err != nil {
		return false, err
	}
	tryLock := func() (locked bool, err error) {
		lock, err := req.Result()
		if err != nil {
			return false, err
		}
		if !lock.Truthy() {
			// lock not yet held
			sharedCount := 0
			if shared {
				sharedCount++
			}
			err := putLock(locks, jsKey, sharedCount)
			if err != nil {
				return false, err
			}
			return true, txn.Commit()
		}

		// lock is held
		sharedCount, err := getSharedCount(lock)
		if err != nil {
			return false, err
		}
		isShared := sharedCount > 0
		if shared {
			sharedCount++
		}
		if shared && isShared { // lock already held by shared: add 1 and return
			err := putLock(locks, jsKey, sharedCount+1)
			if err != nil {
				return false, err
			}
			return true, txn.Commit()
		}
		return false, nil
	}
	var listenErr error
	req.ListenSuccess(ctx, func() {
		locked, listenErr = tryLock()
		if listenErr != nil {
			txn.Abort()
			return
		}
	})
	err = txn.Await(ctx)
	if listenErr != nil {
		return false, listenErr
	}
	if err != nil {
		return false, err
	}
	return locked, nil
}

func putLock(locks *idb.ObjectStore, key js.Value, sharedCount int) error {
	_, err := locks.PutKey(key, js.ValueOf(map[string]interface{}{
		sharedCountField: sharedCount,
	}))
	return err
}

func getSharedCount(lock js.Value) (int, error) {
	jsSharedCount := lock.Get(sharedCountField)
	if jsSharedCount.Type() != js.TypeNumber {
		return 0, errors.New("malformed shared count")
	}
	return jsSharedCount.Int(), nil
}

func drainTicker(ticker *time.Ticker) {
	for {
		select {
		case _, ok := <-ticker.C:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func (f *fileLocker) Unlock(ctx context.Context, filePath string) error {
	txn, err := f.db.Transaction(idb.TransactionReadWrite, locksObjectStore)
	if err != nil {
		return err
	}
	locks, err := txn.ObjectStore(locksObjectStore)
	if err != nil {
		return err
	}
	jsKey := js.ValueOf(filePath)
	req, err := locks.Get(jsKey)
	if err != nil {
		return err
	}
	tryUnlock := func() error {
		lock, err := req.Result()
		if err != nil {
			return err
		}
		sharedCount, err := getSharedCount(lock)
		if err != nil {
			return err
		}
		if sharedCount <= 1 { // is exclusive lock or last shared lock
			_, err := locks.Delete(jsKey)
			if err != nil {
				return err
			}
			return txn.Commit()
		}
		sharedCount--
		err = putLock(locks, jsKey, sharedCount)
		if err != nil {
			return err
		}
		return txn.Commit()
	}
	var listenErr error
	req.ListenSuccess(ctx, func() {
		listenErr = tryUnlock()
		if listenErr != nil {
			txn.Abort()
			return
		}
	})
	err = txn.Await(ctx)
	if listenErr != nil {
		return listenErr
	}
	return err
}
