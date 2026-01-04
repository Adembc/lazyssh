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
	"fmt"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerList struct {
	*tview.List
	servers           []domain.Server
	displayedItems    []*domain.Server
	onSelection       func(domain.Server)
	onSelectionChange func(domain.Server)
	onReturnToSearch  func()
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
		SetTitle(" Servers ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
	sl.List.
		SetSelectedBackgroundColor(tcell.Color24).
		SetSelectedTextColor(tcell.Color255).
		SetHighlightFullLine(true)

	sl.List.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(sl.displayedItems) {
			item := sl.displayedItems[index]
			if item != nil && sl.onSelectionChange != nil {
				sl.onSelectionChange(*item)
			}
		}
	})

	sl.List.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		//nolint:exhaustive // We only handle specific keys and pass through others
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyESC:
			if sl.onReturnToSearch != nil {
				sl.onReturnToSearch()
			}
			return nil
		case tcell.KeyDown:
			return sl.selectNext()
		case tcell.KeyUp:
			return sl.selectPrev()
		}
		return event
	})
}

func (sl *ServerList) UpdateServers(servers []domain.Server) {
	sl.servers = servers
	sl.List.Clear()
	sl.displayedItems = make([]*domain.Server, 0)

	inPinnedSection := false
	firstUnpinned := true
	lastGroup := ""

	hasGroups := false
	for _, s := range servers {
		if s.Group != "" {
			hasGroups = true
			break
		}
	}

	for i := range servers {
		s := servers[i]
		isPinned := !s.PinnedAt.IsZero()

		if isPinned {
			if !inPinnedSection {
				inPinnedSection = true
				if hasGroups {
					sl.List.AddItem("[yellow::b]Pinned[-]", "", 0, nil)
					sl.displayedItems = append(sl.displayedItems, nil)
				}
			}
		} else {
			if inPinnedSection {
				inPinnedSection = false
			}

			if firstUnpinned || s.Group != lastGroup {
				if hasGroups {
					groupName := s.Group
					if groupName == "" {
						groupName = "Ungrouped"
					}
					sl.List.AddItem(fmt.Sprintf("[yellow::b]%s[-]", groupName), "", 0, nil)
					sl.displayedItems = append(sl.displayedItems, nil)
				}
				lastGroup = s.Group
				firstUnpinned = false
			}
		}

		primary, secondary := formatServerLine(servers[i])
		// Indent content
		primary = "  " + primary

		idx := i
		sl.List.AddItem(primary, secondary, 0, func() {
			if sl.onSelection != nil {
				sl.onSelection(sl.servers[idx])
			}
		})
		sl.displayedItems = append(sl.displayedItems, &sl.servers[i])
	}

	if sl.List.GetItemCount() > 0 {
		// Find first selectable item
		firstSelectable := -1
		for i, item := range sl.displayedItems {
			if item != nil {
				firstSelectable = i
				break
			}
		}

		if firstSelectable >= 0 {
			sl.List.SetCurrentItem(firstSelectable)
			if sl.onSelectionChange != nil {
				sl.onSelectionChange(*sl.displayedItems[firstSelectable])
			}
		}
	}
}

func (sl *ServerList) GetSelectedServer() (domain.Server, bool) {
	idx := sl.List.GetCurrentItem()
	if idx >= 0 && idx < len(sl.displayedItems) {
		item := sl.displayedItems[idx]
		if item != nil {
			return *item, true
		}
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

func (sl *ServerList) OnReturnToSearch(fn func()) *ServerList {
	sl.onReturnToSearch = fn
	return sl
}

func (sl *ServerList) selectNext() *tcell.EventKey {
	current := sl.List.GetCurrentItem()
	count := sl.List.GetItemCount()

	if count == 0 {
		return nil
	}

	for i := current + 1; i < count; i++ {
		if i < len(sl.displayedItems) && sl.displayedItems[i] != nil {
			sl.List.SetCurrentItem(i)
			return nil
		}
	}
	return nil
}

func (sl *ServerList) selectPrev() *tcell.EventKey {
	current := sl.List.GetCurrentItem()
	count := sl.List.GetItemCount()

	if count == 0 {
		return nil
	}

	for i := current - 1; i >= 0; i-- {
		if i < len(sl.displayedItems) && sl.displayedItems[i] != nil {
			sl.List.SetCurrentItem(i)
			return nil
		}
	}
	return nil
}
