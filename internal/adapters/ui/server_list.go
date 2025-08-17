// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"github.com/Adembc/lazyssh/internal/core/domain"
	tcell "github.com/gdamore/tcell/v2" // Explicit alias
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
	sl.List.ShowSecondaryText(false)
	sl.List.SetBorder(true).
		SetTitle("Servers").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
	sl.List.
		SetSelectedBackgroundColor(tcell.Color24).
		SetSelectedTextColor(tcell.Color255).
		SetHighlightFullLine(true)

	sl.List.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(sl.servers) && sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[index])
		}
	})
}

func (sl *ServerList) UpdateServers(servers []domain.Server) {
	sl.servers = servers
	sl.List.Clear()

	for i := range servers {
		primary, secondary := formatServerLine(servers[i])
		idx := i
		sl.List.AddItem(primary, secondary, 0, func() {
			if sl.onSelection != nil {
				sl.onSelection(sl.servers[idx])
			}
		})
	}

	if sl.List.GetItemCount() > 0 {
		sl.List.SetCurrentItem(0)
		if sl.onSelectionChange != nil {
			sl.onSelectionChange(sl.servers[0])
		}
	}
}

func (sl *ServerList) GetSelectedServer() (domain.Server, bool) {
	idx := sl.List.GetCurrentItem()
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
