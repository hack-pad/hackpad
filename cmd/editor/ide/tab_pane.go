package ide

import (
	"syscall/js"
)

type Tabber interface {
	Titles() <-chan string
}

type TabPane struct {
	lastTabID        int
	jsValue          js.Value
	tabButtonsParent js.Value
	tabsParent       js.Value
	newTabListener   js.Func
	tabs             []*Tab
	currentTab       int
	makeDefaultTab   TabBuilder
	newTabOptions    TabOptions
}

type TabOptions struct {
	NoFocus bool
}

type TabBuilder func(button, contents js.Value) Tabber

func NewTabPane(newTabOptions TabOptions, makeDefaultTab TabBuilder) *TabPane {
	elem := document.Call("createElement", "div")
	elem.Get("classList").Call("add", "pane")
	elem.Set("innerHTML", `
<nav class="tab-bar">
	<ul class="tab-buttons"></ul>
	<button class="tab-new"></button>
</nav>
<div class="tabs"></div>
`)
	p := &TabPane{
		jsValue:          elem,
		tabButtonsParent: elem.Call("querySelector", ".tab-buttons"),
		tabsParent:       elem.Call("querySelector", ".tabs"),
		tabs:             nil,
		currentTab:       -1,
		makeDefaultTab:   makeDefaultTab,
		newTabOptions:    newTabOptions,
	}
	p.newTabListener = js.FuncOf(func(js.Value, []js.Value) interface{} {
		p.NewTab(newTabOptions, p.makeDefaultTab)
		return nil
	})
	elem.Call("querySelector", ".tab-new").Call("addEventListener", "click", p.newTabListener)
	return p
}

func (p *TabPane) JSValue() js.Value {
	return p.jsValue
}

func (p *TabPane) NewDefaultTab(options TabOptions) Tabber {
	return p.NewTab(options, p.makeDefaultTab)
}

func (p *TabPane) NewTab(options TabOptions, makeTab TabBuilder) Tabber {
	contents := document.Call("createElement", "div")
	contents.Set("className", "tab")
	p.tabsParent.Call("appendChild", contents)

	tabItem := document.Call("createElement", "li")
	tabItem.Set("innerHTML", `<button class="tab-button">New file</button>`)
	button := tabItem.Call("querySelector", "button")
	p.tabButtonsParent.Call("appendChild", tabItem)

	tabber := makeTab(button, contents)
	tab := newTab(p.lastTabID, button, contents, tabber, p.focusID)
	p.lastTabID++
	p.tabs = append(p.tabs, tab)

	if !options.NoFocus {
		p.focusID(tab.id)
	}
	return tabber
}

func (p *TabPane) Focus(index int) {
	if index >= 0 {
		p.focusID(p.tabs[index].id)
	}
}

func (p *TabPane) focusID(id int) {
	if p.currentTab >= 0 {
		p.tabs[p.currentTab].Unfocus()
	}
	for i, tab := range p.tabs {
		if tab.id == id {
			p.currentTab = i
			tab.Focus()
		}
	}
}

func (p *TabPane) Close() {
	p.newTabListener.Release()
}
