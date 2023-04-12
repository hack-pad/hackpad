//go:build js
// +build js

package ide

import (
	"context"
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

type Tab struct {
	id             int
	button         *dom.Element
	buttonListener dom.EventListener
	contents       *dom.Element
	title          *dom.Element
	stopTitlesLoop context.CancelFunc
}

func newTab(id int, button, contents, title *dom.Element, tabber Tabber, focus func(id int)) *Tab {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Tab{
		id:             id,
		button:         button,
		contents:       contents,
		title:          title,
		stopTitlesLoop: cancel,
	}
	go t.watchTitles(ctx, tabber)

	t.buttonListener = func(event js.Value) {
		formElem := t.title.QuerySelector("input")
		if formElem == nil {
			focus(t.id)
		}
	}
	button.AddEventListener("click", t.buttonListener)

	return t
}

func (t *Tab) Focus() {
	t.contents.AddClass("active")
	t.button.AddClass("active")
	firstInput := t.contents.QuerySelector("input, select, textarea")
	if firstInput != nil {
		firstInput.Focus()
	}
}

func (t *Tab) Unfocus() {
	t.contents.RemoveClass("active")
	t.button.RemoveClass("active")
}

func (t *Tab) Close() {
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
				t.title.SetInnerText(title)
			} else {
				t.Close()
			}
		}
	}
}
