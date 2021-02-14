// +build js

package indexeddb

import (
	"syscall/js"

	"github.com/johnstarich/go-wasm/internal/common"
)

type CursorDirection int

const (
	CursorNext CursorDirection = iota
	CursorNextUnique
	CursorPrevious
	CursorPreviousUnique
)

func (d CursorDirection) String() string {
	switch d {
	case CursorNextUnique:
		return "nextunique"
	case CursorPrevious:
		return "previous"
	case CursorPreviousUnique:
		return "previousunique"
	default:
		return "next"
	}
}

type Cursor struct {
	jsCursor js.Value
}

func (c *Cursor) Advance(count uint) (err error) {
	defer common.CatchException(&err)
	c.jsCursor.Call("advance", count)
	return nil
}

func (c *Cursor) Continue() (err error) {
	defer common.CatchException(&err)
	c.jsCursor.Call("continue")
	return nil
}

func (c *Cursor) ContinuePrimaryKey(key, primaryKey js.Value) (err error) {
	defer common.CatchException(&err)
	c.jsCursor.Call("continuePrimaryKey", key, primaryKey)
	return nil
}

func (c *Cursor) Delete() (err error) {
	defer common.CatchException(&err)
	_, err = newRequest(c.jsCursor.Call("delete")).Await()
	return
}

func (c *Cursor) Update(value js.Value) (err error) {
	defer common.CatchException(&err)
	_, err = newRequest(c.jsCursor.Call("update", value)).Await()
	return
}
