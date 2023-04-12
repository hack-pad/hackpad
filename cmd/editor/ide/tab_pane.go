//go:build js
// +build js

package ide

import (
	"syscall/js"

	"github.com/hack-pad/hackpad/cmd/editor/dom"
)

type Tabber interface {
	Titles() <-chan string
}

type TabPane struct {
	*dom.Element
	lastTabID         int
	tabButtonsParent  *dom.Element
	tabsParent        *dom.Element
	tabs              []*Tab
	currentTab        int
	makeDefaultTab    TabBuilder
	newTabOptions     TabOptions
	closedTabListener func(index int)
}

type TabOptions struct {
	NoFocus bool // skips focusing after creating the tab
	NoClose bool // disables the close button
}

type TabBuilder func(id int, title, contents *dom.Element) Tabber

func NewTabPane(newTabOptions TabOptions, makeDefaultTab TabBuilder, closedTab func(index int)) *TabPane {
	elem := dom.New("div")
	elem.AddClass("pane")
	elem.SetInnerHTML(`
<nav class="tab-bar">
	<ul class="tab-buttons"></ul>
	<button class="tab-new" title="new tab"><span class="fa fa-plus"></span></button>
</nav>
<div class="tabs"></div>
`)
	p := &TabPane{
		Element:           elem,
		tabButtonsParent:  elem.QuerySelector(".tab-buttons"),
		tabsParent:        elem.QuerySelector(".tabs"),
		tabs:              nil,
		currentTab:        -1,
		makeDefaultTab:    makeDefaultTab,
		newTabOptions:     newTabOptions,
		closedTabListener: closedTab,
	}
	elem.QuerySelector(".tab-new").AddEventListener("click", func(js.Value) {
		p.NewTab(newTabOptions, p.makeDefaultTab)
	})
	return p
}

func (p *TabPane) NewDefaultTab(options TabOptions) Tabber {
	return p.NewTab(options, p.makeDefaultTab)
}

func (p *TabPane) NewTab(options TabOptions, makeTab TabBuilder) Tabber {
	contents := dom.New("div")
	contents.AddClass("tab")
	p.tabsParent.AppendChild(contents)

	tabItem := dom.New("li")
	tabItem.AddClass("tab-button")
	buttonTemplate := `
<span class="tab-title">New file</span>
`
	if !options.NoClose {
		buttonTemplate += `<button class="tab-close" title="close"><span class="fa fa-times"></span></button>`
	}
	tabItem.SetInnerHTML(buttonTemplate)
	title := tabItem.QuerySelector(".tab-title")
	p.tabButtonsParent.AppendChild(tabItem)

	id := p.lastTabID
	p.lastTabID++
	tabber := makeTab(id, title, contents)
	tab := newTab(id, tabItem, contents, title, tabber, p.focusID)
	p.tabs = append(p.tabs, tab)

	if !options.NoClose {
		closeButton := tabItem.QuerySelector(".tab-close")
		closeButton.AddEventListener("click", func(event js.Value) {
			event.Call("stopPropagation")
			p.closeTabID(tab.id)
		})
	}

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
			return
		}
	}
}

func (p *TabPane) Close() {
}

func (p *TabPane) CloseTab(index int) {
	if index >= 0 {
		p.closeTabID(p.tabs[index].id)
	}
}

func (p *TabPane) closeTabID(id int) {
	var tabIndex int
	var tab *Tab
	for i, t := range p.tabs {
		if t.id == id {
			tabIndex = i
			tab = t
			break
		}
	}
	if tab == nil {
		return
	}

	tab.Close()
	p.tabButtonsParent.Children()[tabIndex].Remove()
	p.tabsParent.Children()[tabIndex].Remove()

	var newTabs []*Tab
	newTabs = append(newTabs, p.tabs[:tabIndex]...)
	newTabs = append(newTabs, p.tabs[tabIndex+1:]...)
	p.tabs = newTabs
	if p.currentTab == len(p.tabs) {
		p.currentTab = len(p.tabs) - 1
	}
	p.Focus(p.currentTab)

	p.closedTabListener(tabIndex)
}
