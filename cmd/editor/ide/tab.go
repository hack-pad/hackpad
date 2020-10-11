package ide

import (
	"context"
	"syscall/js"
)

type Tab struct {
	id             int
	button         js.Value
	buttonListener js.Func
	contents       js.Value
	title          js.Value
	stopTitlesLoop context.CancelFunc
}

func newTab(id int, button, contents, title js.Value, tabber Tabber, focus func(id int)) *Tab {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Tab{
		id:             id,
		button:         button,
		contents:       contents,
		title:          title,
		stopTitlesLoop: cancel,
	}
	go t.watchTitles(ctx, tabber)

	t.buttonListener = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		formElem := t.title.Call("querySelector", "input")
		if formElem.Truthy() {
			return nil
		}
		focus(t.id)
		return nil
	})
	button.Call("addEventListener", "click", t.buttonListener)

	return t
}

func (t *Tab) Focus() {
	t.contents.Get("classList").Call("add", "active")
	t.button.Get("classList").Call("add", "active")
	firstInput := t.contents.Call("querySelector", "input, select, textarea")
	if firstInput.Truthy() {
		firstInput.Call("focus")
	}
}

func (t *Tab) Unfocus() {
	t.contents.Get("classList").Call("remove", "active")
	t.button.Get("classList").Call("remove", "active")
}

func (t *Tab) Close() {
	t.buttonListener.Release()
	t.stopTitlesLoop()
}

func (t *Tab) watchTitles(ctx context.Context, tabber Tabber) {
	titles := tabber.Titles()
	for {
		select {
		case <-ctx.Done():
			return
		case title, ok := <-titles:
			if ok {
				t.title.Set("innerText", title)
			}
		}
	}
}
