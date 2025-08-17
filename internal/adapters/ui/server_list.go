package ui

import (
	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerList struct {
	*tview.List
	servers           []domain.Server
	onSelection       func(domain.Server)
	onSelectionChange func(domain.Server)
}

func NewServerList() *ServerList {
	list := &ServerList{
		List: tview.NewList(),
	}
	list.build()
	return list
}

func (sl *ServerList) build() {
	sl.ShowSecondaryText(false)
	sl.SetBorder(true).
		SetTitle("Servers").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
	sl.
		SetSelectedBackgroundColor(tcell.Color24).
		SetSelectedTextColor(tcell.Color255).
		SetHighlightFullLine(true)

	sl.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(sl.servers) && sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[index])
		}
	})
}

func (sl *ServerList) UpdateServers(servers []domain.Server) {
	sl.servers = servers
	sl.Clear()

	for i := range servers {
		primary, secondary := formatServerLine(servers[i])
		idx := i
		sl.AddItem(primary, secondary, 0, func() {
			if sl.onSelection != nil {
				sl.onSelection(sl.servers[idx])
			}
		})
	}

	if sl.GetItemCount() > 0 {
		sl.SetCurrentItem(0)
		if sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[0])
		}
	}
}

func (sl *ServerList) GetSelectedServer() (domain.Server, bool) {
	idx := sl.GetCurrentItem()
	if idx >= 0 && idx < len(sl.servers) {
		return sl.servers[idx], true
	}
	return domain.Server{}, false
}

func (sl *ServerList) OnSelection(fn func(server domain.Server)) *ServerList {
	sl.onSelection = fn
	return sl
}

func (sl *ServerList) OnSelectionChange(fn func(server domain.Server)) *ServerList {
	sl.onSelectionChange = fn
	return sl
}
