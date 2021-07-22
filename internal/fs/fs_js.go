// +build js

package fs

import (
	"context"

	"github.com/hack-pad/go-indexeddb/idb"
	"github.com/hack-pad/hackpadfs/indexeddb"
)

type persistFs struct {
	*indexeddb.FS
}

func newPersistDB(name string, relaxedDurability bool, shouldCache ShouldCacher) (*persistFs, error) {
	durability := idb.DurabilityDefault
	if relaxedDurability {
		durability = idb.DurabilityRelaxed
	}
	fs, err := indexeddb.NewFS(context.Background(), name, indexeddb.Options{
		TransactionDurability: durability,
	})
	return &persistFs{fs}, err
}
