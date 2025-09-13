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
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// =============================================================================
// Event Handlers (handle user input/events)
// =============================================================================

func (t *tui) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	// Don't handle global keys when search has focus
	if t.app.GetFocus() == t.searchBar {
		return event
	}

	switch event.Rune() {
	case 'q':
		t.handleQuit()
		return nil
	case '/':
		t.handleSearchToggle()
		return nil
	case 'a':
		t.handleServerAdd()
		return nil
	case 'e':
		t.handleServerEdit()
		return nil
	case 'd':
		t.handleServerDelete()
		return nil
	case 'p':
		t.handleServerPin()
		return nil
	case 's':
		t.handleSortToggle()
		return nil
	case 'S':
		t.handleSortReverse()
		return nil
	case 'c':
		t.handleCopyCommand()
		return nil
	case 'g':
		t.handlePingSelected()
		return nil
	case 'G':
		t.handlePingAll()
		return nil
	case 'r':
		t.handleRefreshBackground()
		return nil
	case 't':
		t.handleTagsEdit()
		return nil
	case 'j':
		t.handleNavigateDown()
		return nil
	case 'k':
		t.handleNavigateUp()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		t.handleServerConnect()
		return nil
	}

	return event
}

func (t *tui) handleQuit() {
	t.app.Stop()
}

func (t *tui) handleServerPin() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		pinned := server.PinnedAt.IsZero()
		_ = t.serverService.SetPinned(server.Alias, pinned)
		t.refreshServerList()
	}
}

func (t *tui) handleSortToggle() {
	t.sortMode = t.sortMode.ToggleField()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.refreshServerList()
}

func (t *tui) handleSortReverse() {
	t.sortMode = t.sortMode.Reverse()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.refreshServerList()
}

func (t *tui) handleCopyCommand() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		cmd := BuildSSHCommand(server)
		if err := clipboard.WriteAll(cmd); err == nil {
			t.showStatusTemp("Copied: " + cmd)
		} else {
			t.showStatusTemp("Failed to copy to clipboard")
		}
	}
}

func (t *tui) handleTagsEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showEditTagsForm(server)
	}
}

func (t *tui) handleNavigateDown() {
	if t.app.GetFocus() == t.serverList {
		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()
		if currentIdx < itemCount-1 {
			t.serverList.SetCurrentItem(currentIdx + 1)
		} else {
			t.serverList.SetCurrentItem(0)
		}
	}
}

func (t *tui) handleNavigateUp() {
	if t.app.GetFocus() == t.serverList {
		currentIdx := t.serverList.GetCurrentItem()
		if currentIdx > 0 {
			t.serverList.SetCurrentItem(currentIdx - 1)
		} else {
			t.serverList.SetCurrentItem(t.serverList.GetItemCount() - 1)
		}
	}
}

func (t *tui) handleSearchInput(query string) {
	filtered, _ := t.serverService.ListServers(query)
	sortServersForUI(filtered, t.sortMode)
	t.serverList.UpdateServers(filtered)
	if len(filtered) == 0 {
		t.details.ShowEmpty()
	}
}

func (t *tui) handleSearchToggle() {
	t.showSearchBar()
}

func (t *tui) handleServerConnect() {
	if server, ok := t.serverList.GetSelectedServer(); ok {

		t.app.Suspend(func() {
			_ = t.serverService.SSH(server.Alias)
		})
		t.refreshServerList()
	}
}

func (t *tui) handleServerSelectionChange(server domain.Server) {
	t.details.UpdateServer(server)
}

func (t *tui) handleServerAdd() {
	form := NewServerForm(ServerFormAdd, nil).
		OnSave(t.handleServerSave).
		OnCancel(t.handleFormCancel)
	t.app.SetRoot(form, true)
}

func (t *tui) handleServerEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		form := NewServerForm(ServerFormEdit, &server).
			OnSave(t.handleServerSave).
			OnCancel(t.handleFormCancel)
		t.app.SetRoot(form, true)
	}
}

func (t *tui) handleServerSave(server domain.Server, original *domain.Server) {
	var err error
	if original != nil {
		// Edit mode
		err = t.serverService.UpdateServer(*original, server)
	} else {
		// Add mode
		err = t.serverService.AddServer(server)
	}
	if err != nil {
		// Stay on form; show a small modal with the error
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Save failed: %v", err)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) { t.handleModalClose() })
		t.app.SetRoot(modal, true)
		return
	}

	t.refreshServerList()
	t.handleFormCancel()
}

func (t *tui) handleServerDelete() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showDeleteConfirmModal(server)
	}
}

func (t *tui) handleFormCancel() {
	t.returnToMain()
}

const (
	statusUp       = "up"
	statusDown     = "down"
	statusChecking = "checking"
)

func (t *tui) handlePingSelected() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias

		// Set checking status
		server.PingStatus = statusChecking
		t.pingStatuses[alias] = server
		t.updateServerListWithPingStatus()

		t.showStatusTemp(fmt.Sprintf("Pinging %s…", alias))
		go func() {
			up, dur, err := t.serverService.Ping(server)
			t.app.QueueUpdateDraw(func() {
				// Update ping status
				if ps, ok := t.pingStatuses[alias]; ok {
					if err != nil || !up {
						ps.PingStatus = statusDown
						ps.PingLatency = 0
						t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN", alias), "#FF6B6B")
					} else {
						ps.PingStatus = statusUp
						ps.PingLatency = dur
						t.showStatusTempColor(fmt.Sprintf("Ping %s: UP (%s)", alias, dur), "#A0FFA0")
					}
					t.pingStatuses[alias] = ps
					t.updateServerListWithPingStatus()
				}
			})
		}()
	}
}

func (t *tui) handleModalClose() {
	t.returnToMain()
}

// handleRefreshBackground refreshes the server list in the background without leaving the current screen.
// It preserves the current search query and selection, shows transient status, and avoids concurrent runs.
func (t *tui) handleRefreshBackground() {
	currentIdx := t.serverList.GetCurrentItem()
	query := ""
	if t.searchVisible {
		query = t.searchBar.InputField.GetText()
	}

	t.showStatusTemp("Refreshing…")

	go func(prevIdx int, q string) {
		servers, err := t.serverService.ListServers(q)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showStatusTempColor(fmt.Sprintf("Refresh failed: %v", err), "#FF6B6B")
			})
			return
		}
		sortServersForUI(servers, t.sortMode)
		t.app.QueueUpdateDraw(func() {
			t.serverList.UpdateServers(servers)
			// Try to restore selection if still valid
			if prevIdx >= 0 && prevIdx < t.serverList.List.GetItemCount() {
				t.serverList.SetCurrentItem(prevIdx)
				if srv, ok := t.serverList.GetSelectedServer(); ok {
					t.details.UpdateServer(srv)
				}
			}
			t.showStatusTemp(fmt.Sprintf("Refreshed %d servers", len(servers)))
		})
	}(currentIdx, query)
}

// =============================================================================
// UI Display Functions (show UI elements/modals)
// =============================================================================

func (t *tui) showSearchBar() {
	t.left.Clear()
	t.left.AddItem(t.searchBar, 3, 0, true)
	t.left.AddItem(t.serverList, 0, 1, false)
	t.app.SetFocus(t.searchBar)
	t.searchVisible = true
}

func (t *tui) showDeleteConfirmModal(server domain.Server) {
	msg := fmt.Sprintf("Delete server %s (%s@%s:%d)?\n\nThis action cannot be undone.",
		server.Alias, server.User, server.Host, server.Port)

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Cancel", "Confirm"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				_ = t.serverService.DeleteServer(server)
				t.refreshServerList()
			}
			t.handleModalClose()
		})

	t.app.SetRoot(modal, true)
}

func (t *tui) showEditTagsForm(server domain.Server) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf("Edit Tags: %s", server.Alias)).
		SetTitleAlign(tview.AlignLeft)

	defaultTags := strings.Join(server.Tags, ", ")
	form.AddInputField("Tags (comma):", defaultTags, 40, nil, nil)

	form.AddButton("Save", func() {
		text := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		var tags []string

		for _, part := range strings.Split(text, ",") {
			if s := strings.TrimSpace(part); s != "" {
				tags = append(tags, s)
			}
		}

		newServer := server
		newServer.Tags = tags
		_ = t.serverService.UpdateServer(server, newServer)
		// Refresh UI and go back
		t.refreshServerList()
		t.returnToMain()
		t.showStatusTemp("Tags updated")
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	t.app.SetFocus(form)
}

// =============================================================================
// UI State Management (hide UI elements)
// =============================================================================

func (t *tui) hideSearchBar() {
	t.left.Clear()
	t.left.AddItem(t.hintBar, 1, 0, false)
	t.left.AddItem(t.serverList, 0, 1, true)
	t.app.SetFocus(t.serverList)
	t.searchVisible = false
}

// =============================================================================
// Internal Operations (perform actual work)
// =============================================================================

func (t *tui) refreshServerList() {
	query := ""
	if t.searchVisible {
		query = t.searchBar.InputField.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)
	sortServersForUI(filtered, t.sortMode)
	t.serverList.UpdateServers(filtered)
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}

func (t *tui) updateServerListWithPingStatus() {
	// Get current server list
	query := ""
	if t.searchVisible {
		query = t.searchBar.InputField.GetText()
	}
	servers, _ := t.serverService.ListServers(query)
	sortServersForUI(servers, t.sortMode)

	// Update ping status for each server
	for i := range servers {
		if ps, ok := t.pingStatuses[servers[i].Alias]; ok {
			servers[i].PingStatus = ps.PingStatus
			servers[i].PingLatency = ps.PingLatency
		}
	}

	t.serverList.UpdateServers(servers)
}

func (t *tui) handlePingAll() {
	query := ""
	if t.searchVisible {
		query = t.searchBar.InputField.GetText()
	}
	servers, err := t.serverService.ListServers(query)
	if err != nil {
		t.showStatusTempColor(fmt.Sprintf("Failed to get servers: %v", err), "#FF6B6B")
		return
	}

	if len(servers) == 0 {
		t.showStatusTemp("No servers to ping")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Pinging all %d servers…", len(servers)))

	// Clear existing statuses
	t.pingStatuses = make(map[string]domain.Server)

	// Set all servers to checking status
	for _, server := range servers {
		s := server
		s.PingStatus = statusChecking
		t.pingStatuses[s.Alias] = s
	}
	t.updateServerListWithPingStatus()

	// Ping all servers concurrently
	for _, server := range servers {
		go func(srv domain.Server) {
			up, dur, err := t.serverService.Ping(srv)
			t.app.QueueUpdateDraw(func() {
				if ps, ok := t.pingStatuses[srv.Alias]; ok {
					if err != nil || !up {
						ps.PingStatus = statusDown
						ps.PingLatency = 0
					} else {
						ps.PingStatus = statusUp
						ps.PingLatency = dur
					}
					t.pingStatuses[srv.Alias] = ps
					t.updateServerListWithPingStatus()
				}
			})
		}(server)
	}

	// Show completion status after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		t.app.QueueUpdateDraw(func() {
			upCount := 0
			downCount := 0
			for _, ps := range t.pingStatuses {
				if ps.PingStatus == statusUp {
					upCount++
				} else if ps.PingStatus == statusDown {
					downCount++
				}
			}
			t.showStatusTempColor(fmt.Sprintf("Ping completed: %d UP, %d DOWN", upCount, downCount), "#A0FFA0")
		})
	}()
}

// showStatusTemp displays a temporary message in the status bar (default green) and then restores the default text.
func (t *tui) showStatusTemp(msg string) {
	if t.statusBar == nil {
		return
	}
	t.showStatusTempColor(msg, "#A0FFA0")
}

// showStatusTempColor displays a temporary colored message in the status bar and restores default text after 2s.
func (t *tui) showStatusTempColor(msg string, color string) {
	if t.statusBar == nil {
		return
	}
	t.statusBar.SetText("[" + color + "]" + msg + "[-]")
	time.AfterFunc(2*time.Second, func() {
		if t.app != nil {
			t.app.QueueUpdateDraw(func() {
				if t.statusBar != nil {
					t.statusBar.SetText(DefaultStatusText())
				}
			})
		}
	})
}
