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
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerList struct {
	*tview.List
	servers           []domain.Server
	displayedItems    []*domain.Server
	displayedHeaders  []string
	collapsedGroups   map[string]bool
	onSelection       func(domain.Server)
	onSelectionChange func(domain.Server)
	onReturnToSearch  func()
	onGroupAction     func(groupName string, action string)
}

func NewServerList() *ServerList {
	list := &ServerList{
		List:            tview.NewList(),
		collapsedGroups: make(map[string]bool),
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
		case tcell.KeyEnter, tcell.KeyRune:
			isSpace := event.Key() == tcell.KeyRune && event.Rune() == ' '
			isEnter := event.Key() == tcell.KeyEnter
			isMenu := event.Key() == tcell.KeyRune && event.Rune() == 'm'

			idx := sl.List.GetCurrentItem()
			if idx >= 0 && idx < len(sl.displayedHeaders) {
				groupName := sl.displayedHeaders[idx]

				// Handle Group Actions
				if groupName != "" {
					if isSpace || isEnter {
						// Toggle Collapse
						sl.collapsedGroups[groupName] = !sl.collapsedGroups[groupName]
						sl.UpdateServers(sl.servers)

						// Try to find the header again to restore selection
						newIdx := -1
						for i, h := range sl.displayedHeaders {
							if h == groupName {
								newIdx = i
								break
							}
						}
						if newIdx >= 0 {
							sl.List.SetCurrentItem(newIdx)
						}
						return nil
					} else if isMenu {
						// Trigger Context Menu Action
						sl.showGroupContextMenu(groupName)
						return nil
					}
				}
			}
		}
		return event
	})
}

func (sl *ServerList) UpdateServers(servers []domain.Server) {
	sl.servers = servers
	sl.List.Clear()
	sl.displayedItems = make([]*domain.Server, 0)
	sl.displayedHeaders = make([]string, 0)

	inPinnedSection := false

	// Helper to track nested groups
	lastGroupParts := []string{}
	hasGroups := false
	for _, s := range servers {
		if s.Group != "" {
			hasGroups = true
			break
		}
	}

	addHeader := func(fullPath string, name string, depth int) {
		isCollapsed := sl.collapsedGroups[fullPath]
		icon := "[-]"
		if isCollapsed {
			icon = "[+]"
		}
		indent := strings.Repeat("  ", depth)
		sl.List.AddItem(fmt.Sprintf("%s[yellow::b]%s %s[-]", indent, icon, name), "", 0, nil)
		sl.displayedItems = append(sl.displayedItems, nil)
		sl.displayedHeaders = append(sl.displayedHeaders, fullPath)
	}

	for i := range servers {
		s := servers[i]
		isPinned := !s.PinnedAt.IsZero()

		if isPinned {
			if !inPinnedSection {
				inPinnedSection = true
				if hasGroups {
					addHeader("Pinned", "Pinned", 0)
				}
				// Reset group context when entering pinned
				lastGroupParts = []string{}
			}
			if hasGroups && sl.collapsedGroups["Pinned"] {
				continue
			}
		} else {
			if inPinnedSection {
				inPinnedSection = false
				// Reset group context when leaving pinned
				lastGroupParts = []string{}
			}

			if hasGroups {
				currentGroup := s.Group
				if currentGroup == "" {
					currentGroup = "Ungrouped"
				}

				currentParts := strings.Split(currentGroup, "/")

				// Calculate common prefix with previous server's group
				commonLen := 0
				for j := 0; j < len(lastGroupParts) && j < len(currentParts); j++ {
					if lastGroupParts[j] == currentParts[j] {
						commonLen++
					} else {
						break
					}
				}

				// Determine visibility based on parent groups
				serverVisible := true
				fullPath := ""

				// Check visibility and render headers for divergent parts
				for j, part := range currentParts {
					if j > 0 {
						fullPath += "/"
					}
					fullPath += part

					// Check if any parent up to this point is collapsed
					// But we only care if a *parent* is collapsed to hide *this* header.
					// The header itself being collapsed affects its children.

					// Wait, we need to check if the PARENT of the current header is collapsed
					// to decide if we show THIS header.
					parentPath := ""
					if j > 0 {
						parentPath = fullPath[:strings.LastIndex(fullPath, "/")]
					}

					parentCollapsed := false
					if parentPath != "" && sl.collapsedGroups[parentPath] {
						parentCollapsed = true
					} else if j == 0 && inPinnedSection {
						// Should not happen as we handle pinned separately, but conceptually
					}

					// If parent is collapsed, we stop everything down this path
					if parentCollapsed {
						serverVisible = false
						break
					}

					// Render header if it's new (divergent from last)
					if j >= commonLen {
						addHeader(fullPath, part, j)
					}

					// Check if THIS group is collapsed (affects children and server)
					if sl.collapsedGroups[fullPath] {
						serverVisible = false
					}
				}

				lastGroupParts = currentParts

				if !serverVisible {
					continue
				}

				// Update indent for server
				primary, secondary := formatServerLine(servers[i])
				indent := strings.Repeat("  ", len(currentParts)+1)
				primary = indent + primary

				idx := i
				sl.List.AddItem(primary, secondary, 0, func() {
					if sl.onSelection != nil {
						sl.onSelection(sl.servers[idx])
					}
				})
				sl.displayedItems = append(sl.displayedItems, &sl.servers[i])
				sl.displayedHeaders = append(sl.displayedHeaders, "")
				continue
			}
		}

		// Fallback for no groups or Pinned items rendering
		primary, secondary := formatServerLine(servers[i])
		primary = "  " + primary

		idx := i
		sl.List.AddItem(primary, secondary, 0, func() {
			if sl.onSelection != nil {
				sl.onSelection(sl.servers[idx])
			}
		})
		sl.displayedItems = append(sl.displayedItems, &sl.servers[i])
		sl.displayedHeaders = append(sl.displayedHeaders, "")
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
		// If no items found (all collapsed), select first header
		if firstSelectable == -1 {
			firstSelectable = 0
		}

		if firstSelectable >= 0 {
			sl.List.SetCurrentItem(firstSelectable)
			if sl.onSelectionChange != nil && sl.displayedItems[firstSelectable] != nil {
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

func (sl *ServerList) OnGroupAction(fn func(groupName string, action string)) *ServerList {
	sl.onGroupAction = fn
	return sl
}

func (sl *ServerList) showGroupContextMenu(groupName string) {
	// Trigger the callback to let the parent (TUI) handle the menu display
	// We pass "menu" action to indicate that a context menu is requested
	if sl.onGroupAction != nil {
		sl.onGroupAction(groupName, "menu")
	}
}

func (sl *ServerList) selectNext() *tcell.EventKey {
	current := sl.List.GetCurrentItem()
	count := sl.List.GetItemCount()

	if count == 0 {
		return nil
	}

	for i := current + 1; i < count; i++ {
		if i < len(sl.displayedItems) {
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
		if i < len(sl.displayedItems) {
			sl.List.SetCurrentItem(i)
			return nil
		}
	}
	return nil
}
