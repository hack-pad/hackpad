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

func newPersistDB(name string, shouldCache ShouldCacher) (*persistFs, error) {
	fs, err := indexeddb.NewFS(context.Background(), name, idb.Global())
	return &persistFs{fs}, err
}
