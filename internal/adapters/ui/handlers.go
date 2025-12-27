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
	"os"
	"path/filepath"
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
const (
	ForwardTypeLocal   = "Local"
	ForwardTypeRemote  = "Remote"
	ForwardTypeDynamic = "Dynamic"

	ForwardModeOnlyForward = "Only forward"
	ForwardModeForwardSSH  = "Forward + SSH"
)

//nolint:gocyclo // High complexity due to many key bindings - acceptable for this handler
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
		t.handleSearchFocus()
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
	case 'r':
		t.handleRefreshBackground()
		return nil
	case 't':
		t.handleTagsEdit()
		return nil
	case 'f':
		t.handlePortForward()
		return nil
	case 'x':
		t.handleStopForwarding()
		return nil
	case 'j':
		t.handleNavigateDown()
		return nil
	case 'k':
		t.handleNavigateUp()
		return nil
	case 'l':
		if t.currentPanel == 1 {
			t.handleLoadKey()
			return nil
		}
		// Also handle 'l' key for loading server's SSH key
		if t.currentPanel == 0 {
			t.handleLoadServerKey()
			return nil
		}
	case 'u':
		if t.currentPanel == 1 {
			t.handleUnloadKey()
			return nil
		}
		// Also handle 'u' key for unloading server's SSH key
		if t.currentPanel == 0 {
			t.handleUnloadServerKey()
			return nil
		}
	case 'G':
		t.handleGitSSHSetup()
		return nil
	}

	// Handle 'C' key (Shift+c) for editing SSH key comment
	if event.Rune() == 'C' {
		if t.currentPanel == 1 {
			t.handleEditKeyComment()
			return nil
		}
		if t.currentPanel == 0 {
			t.handleEditServerKeyComment()
			return nil
		}
	}

	if event.Key() == tcell.KeyEnter {
		if t.currentPanel != 0 {
			// Enter key on SSH Keys panels - no action for now
			return nil
		}
		t.handleServerConnect()
		return nil
	}

	if event.Key() == tcell.KeyTab {
		t.handleTabSwitch()
		return nil
	}

	if event.Key() == tcell.KeyBacktab {
		t.handleTabSwitch()
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
	var currentIdx, itemCount int

	switch t.currentPanel {
	case 0:
		// Navigate in Servers panel
		currentIdx = t.serverList.GetCurrentItem()
		itemCount = t.serverList.GetItemCount()
		if currentIdx < itemCount-1 {
			t.serverList.SetCurrentItem(currentIdx + 1)
		} else {
			t.serverList.SetCurrentItem(0)
		}
	case 1:
		// Navigate in SSH Agent panel
		currentIdx = t.sshAgentKeysList.GetCurrentItem()
		itemCount = t.sshAgentKeysList.GetItemCount()
		if currentIdx < itemCount-1 {
			t.sshAgentKeysList.SetCurrentItem(currentIdx + 1)
		} else {
			t.sshAgentKeysList.SetCurrentItem(0)
		}
	}
}

func (t *tui) handleNavigateUp() {
	var currentIdx, itemCount int

	switch t.currentPanel {
	case 0:
		// Navigate in Servers panel
		currentIdx = t.serverList.GetCurrentItem()
		itemCount = t.serverList.GetItemCount()
		if currentIdx > 0 {
			t.serverList.SetCurrentItem(currentIdx - 1)
		} else {
			t.serverList.SetCurrentItem(itemCount - 1)
		}
	case 1:
		// Navigate in SSH Agent panel
		currentIdx = t.sshAgentKeysList.GetCurrentItem()
		itemCount = t.sshAgentKeysList.GetItemCount()
		if currentIdx > 0 {
			t.sshAgentKeysList.SetCurrentItem(currentIdx - 1)
		} else {
			t.sshAgentKeysList.SetCurrentItem(itemCount - 1)
		}
	}
}

func (t *tui) handleTabSwitch() {
	// Cycle through panels: 0 (Servers) → 1 (SSH Agent) → 0
	t.currentPanel = (t.currentPanel + 1) % 2

	// Update focus to the new panel
	switch t.currentPanel {
	case 0:
		t.app.SetFocus(t.serverList)
		if server, ok := t.serverList.GetSelectedServer(); ok {
			t.serverDetails.UpdateServer(server)
		}
	case 1:
		t.app.SetFocus(t.sshAgentKeysList)
		if key, ok := t.sshAgentKeysList.GetSelectedKey(); ok {
			t.sshKeyDetails.UpdateKey(key)
		}
	}

	t.updatePanelBorders()
	t.updateRightPanel()
}

func (t *tui) handlePanelClick(panelIndex int) {
	// If clicking on the already-focused panel, do nothing
	if t.currentPanel == panelIndex {
		return
	}

	// Switch focus to the clicked panel
	t.currentPanel = panelIndex

	switch t.currentPanel {
	case 0:
		t.app.SetFocus(t.serverList)
		if server, ok := t.serverList.GetSelectedServer(); ok {
			t.serverDetails.UpdateServer(server)
		}
	case 1:
		t.app.SetFocus(t.sshAgentKeysList)
		if key, ok := t.sshAgentKeysList.GetSelectedKey(); ok {
			t.sshKeyDetails.UpdateKey(key)
		}
	}

	t.updatePanelBorders()
	t.updateRightPanel()
}

func (t *tui) handleSearchInput(query string) {
	filtered, _ := t.serverService.ListServers(query)
	sortServersForUI(filtered, t.sortMode)
	t.serverList.UpdateServers(filtered)
	if len(filtered) == 0 {
		t.serverDetails.ShowEmpty()
	}
}

func (t *tui) handleSearchFocus() {
	if t.app != nil && t.searchBar != nil {
		t.app.SetFocus(t.searchBar)
	}
}

func (t *tui) handleSearchNavigate(direction int) {
	if t.serverList != nil {
		t.app.SetFocus(t.serverList)

		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()

		if itemCount == 0 {
			return
		}

		if direction > 0 {
			if currentIdx < itemCount-1 {
				t.serverList.SetCurrentItem(currentIdx + 1)
			} else {
				t.serverList.SetCurrentItem(0)
			}
		} else {
			if currentIdx > 0 {
				t.serverList.SetCurrentItem(currentIdx - 1)
			} else {
				t.serverList.SetCurrentItem(itemCount - 1)
			}
		}

		if server, ok := t.serverList.GetSelectedServer(); ok {
			t.serverDetails.UpdateServer(server)
		}
	}
}

func (t *tui) handleReturnToSearch() {
	if t.searchBar != nil {
		t.app.SetFocus(t.searchBar)
	}
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
	t.serverDetails.UpdateServer(server)
}

func (t *tui) handleSSHKeySelectionChange(key domain.SSHKey) {
	if t.sshKeyDetails != nil {
		t.sshKeyDetails.UpdateKey(key)
	}
}

func (t *tui) handleServerAdd() {
	form := NewServerForm(ServerFormAdd, nil).
		SetApp(t.app).
		SetVersionInfo(t.version, t.commit).
		OnSave(t.handleServerSave).
		OnCancel(t.handleFormCancel)
	t.app.SetRoot(form, true)
}

func (t *tui) handleServerEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		form := NewServerForm(ServerFormEdit, &server).
			SetApp(t.app).
			SetVersionInfo(t.version, t.commit).
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

func (t *tui) handlePingSelected() {
	if server, ok := t.serverList.GetSelectedServer(); ok {

		alias := server.Alias

		t.showStatusTemp(fmt.Sprintf("Pinging %s…", alias))
		go func() {
			up, dur, err := t.serverService.Ping(server)
			t.app.QueueUpdateDraw(func() {
				if err != nil {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN (%v)", alias, err), "#FF6B6B")
					return
				}
				if up {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: UP (%s)", alias, dur), "#A0FFA0")
				} else {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN", alias), "#FF6B6B")
				}
			})
		}()
	}
}

func (t *tui) handleModalClose() {
	t.updateGitInfoPanel()
	t.returnToMain()
}

// handleRefreshBackground refreshes the server list in the background without leaving the current screen.
// It preserves the current search query and selection, shows transient status, and avoids concurrent runs.
func (t *tui) handleRefreshBackground() {
	currentIdx := t.serverList.GetCurrentItem()
	query := ""
	if t.searchBar != nil {
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
					t.serverDetails.UpdateServer(srv)
				}
			}
			t.showStatusTemp(fmt.Sprintf("Refreshed %d servers", len(servers)))
		})
	}(currentIdx, query)
}

// =============================================================================
// UI Display Functions (show UI elements/modals)
// =============================================================================

func (t *tui) showDeleteConfirmModal(server domain.Server) {
	msg := fmt.Sprintf("Delete server %s (%s@%s:%d)?\n\nThis action cannot be undone.",
		server.Alias, server.User, server.Host, server.Port)

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"[yellow]C[-]ancel", "[yellow]D[-]elete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				_ = t.serverService.DeleteServer(server)
				t.refreshServerList()
			}
			t.handleModalClose()
		})

	// Add keyboard shortcuts for the modal
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'c', 'C':
			// Cancel
			t.handleModalClose()
			return nil
		case 'd', 'D':
			// Delete
			_ = t.serverService.DeleteServer(server)
			t.refreshServerList()
			t.handleModalClose()
			return nil
		}
		// ESC key already handled by default modal behavior
		return event
	})

	t.app.SetRoot(modal, true)
}

func (t *tui) showEditTagsForm(server domain.Server) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Tags: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

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
	toFocus := form
	t.app.SetFocus(toFocus)
}

func (t *tui) handlePortForward() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showPortForwardForm(server)
	}
}

func (t *tui) showPortForwardForm(server domain.Server) {
	typeChoices := []string{ForwardTypeLocal, ForwardTypeRemote, ForwardTypeDynamic}
	modeChoices := []string{ForwardModeOnlyForward, ForwardModeForwardSSH}

	currentTypeIdx := 0
	currentModeIdx := 0
	portVal := ""
	hostVal := "localhost"
	hostPortVal := ""
	bindAddrVal := ""

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Port Forwarding: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	dd := tview.NewDropDown()
	hostField := tview.NewInputField()
	hostPortField := tview.NewInputField()
	portField := tview.NewInputField()
	bindAddrField := tview.NewInputField()

	dd.SetOptions(typeChoices, func(text string, index int) {
		currentTypeIdx = index
		// Toggle fields when switching type
		isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
		if isDynamic {
			hostField.SetText("").SetDisabled(true)
			hostPortField.SetText("").SetDisabled(true)
		} else {
			hostField.SetDisabled(false)
			hostPortField.SetDisabled(false)
		}
	})
	dd.SetCurrentOption(currentTypeIdx)
	form.AddFormItem(dd.SetLabel("Type"))

	portField.SetLabel("Port").SetText(portVal).SetFieldWidth(8).SetChangedFunc(func(text string) { portVal = strings.TrimSpace(text) })
	form.AddFormItem(portField)

	hostField.SetLabel("Host").SetText(hostVal).SetFieldWidth(40).SetChangedFunc(func(text string) { hostVal = strings.TrimSpace(text) })
	form.AddFormItem(hostField)

	hostPortField.SetLabel("Host Port").SetText(hostPortVal).SetFieldWidth(8).SetChangedFunc(func(text string) { hostPortVal = strings.TrimSpace(text) })
	form.AddFormItem(hostPortField)

	bindAddrField.SetLabel("Bind Address (optional)").SetText(bindAddrVal).SetFieldWidth(40).SetChangedFunc(func(text string) { bindAddrVal = strings.TrimSpace(text) })
	form.AddFormItem(bindAddrField)

	mode := tview.NewDropDown().SetOptions(modeChoices, func(text string, index int) { currentModeIdx = index })
	mode.SetCurrentOption(currentModeIdx)
	form.AddFormItem(mode.SetLabel("Mode"))

	isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
	if isDynamic {
		hostField.SetText("").SetDisabled(true)
		hostPortField.SetText("").SetDisabled(true)
	}

	form.AddButton("Start", func() {
		if err := validatePort(portVal); err != nil {
			t.showStatusTempColor("Invalid port: "+err.Error(), "#FF6B6B")
			return
		}
		if bindAddrVal != "" {
			if err := validateBindAddress(bindAddrVal); err != nil {
				t.showStatusTempColor("Invalid bind address: "+err.Error(), "#FF6B6B")
				return
			}
		}

		ft := typeChoices[currentTypeIdx]
		var args []string
		if ft == ForwardTypeDynamic {
			spec := portVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + portVal
			}
			args = append(args, "-D", spec)
		} else {
			if err := validateHost(hostVal); err != nil {
				t.showStatusTempColor("Invalid host: "+err.Error(), "#FF6B6B")
				return
			}
			if err := validatePort(hostPortVal); err != nil {
				t.showStatusTempColor("Invalid host port: "+err.Error(), "#FF6B6B")
				return
			}
			spec := portVal + ":" + hostVal + ":" + hostPortVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + spec
			}
			if ft == ForwardTypeLocal {
				args = append(args, "-L", spec)
			} else {
				args = append(args, "-R", spec)
			}
		}

		onlyForward := modeChoices[currentModeIdx] == ForwardModeOnlyForward
		alias := server.Alias
		if onlyForward {
			t.returnToMain()
			t.showStatusTemp("Starting port forward…")
			go func() {
				pid, err := t.serverService.StartForward(alias, args)
				t.app.QueueUpdateDraw(func() {
					if err != nil {
						t.showStatusTempColor("Forward failed: "+err.Error(), "#FF6B6B")
					} else {
						t.refreshServerList()
						t.showStatusTemp(fmt.Sprintf("Port forwarding started (pid %d)", pid))
					}
				})
			}()
			return
		}

		t.app.Suspend(func() {
			_ = t.serverService.SSHWithArgs(alias, args)
		})
		t.returnToMain()
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	t.app.SetFocus(form)
}

// =============================================================================
// UI State Management (hide UI elements)
// =============================================================================

// blurSearchBar moves focus back to the server list without changing layout.
func (t *tui) blurSearchBar() {
	if t.app != nil && t.serverList != nil {
		t.app.SetFocus(t.serverList)
	}
}

// =============================================================================
// Internal Operations (perform actual work)
// =============================================================================

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

// Stop any active port forwarding for the selected server.
func (t *tui) handleStopForwarding() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias
		go func() {
			err := t.serverService.StopForwarding(alias)
			t.app.QueueUpdateDraw(func() {
				if err != nil {
					t.showStatusTempColor("Failed to stop forwarding: "+err.Error(), "#FF6B6B")
				} else {
					t.showStatusTemp("Stopped forwarding for " + alias)
				}
				t.refreshServerList()
			})
		}()
	}
}

func (t *tui) handleLoadKey() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get selected key from SSH Agent panel
	key, ok := t.sshAgentKeysList.GetSelectedKey()
	if !ok {
		t.showStatusTempColor("No SSH key selected", "#FF6B6B")
		return
	}

	if key.LoadedInAgent {
		t.showStatusTempColor("Key already loaded in ssh-agent", "#FFA500")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Loading key %s...", key.Name))

	// LoadKeyToAgent needs user interaction for passphrase
	// Suspend is blocking, so we can do this synchronously
	keyPath := key.Path
	keyName := key.Name

	t.app.Suspend(func() {
		err := t.gitService.LoadKeyToAgent(keyPath)
		if err != nil {
			fmt.Printf("\nFailed to load key: %v\n", err)
			fmt.Print("Press Enter to continue...")
			var dummy string
			_, _ = fmt.Scanln(&dummy)
		}
	})

	// Refresh SSH keys list after load (back in main thread)
	t.refreshSSHKeysList()
	t.showStatusTemp(fmt.Sprintf("Key %s loaded successfully", keyName))
}

func (t *tui) handleUnloadKey() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get selected key from SSH Agent panel
	key, ok := t.sshAgentKeysList.GetSelectedKey()
	if !ok {
		t.showStatusTempColor("No SSH key selected", "#FF6B6B")
		return
	}

	if !key.LoadedInAgent {
		t.showStatusTempColor("Key not loaded in ssh-agent", "#FFA500")
		return
	}

	if key.PublicKeyLine == "" {
		t.showStatusTempColor("Key has no public key line, cannot unload", "#FF6B6B")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Unloading key %s...", key.Name))

	go func(publicKeyLine string, keyName string) {
		err := t.gitService.UnloadKeyFromAgent(publicKeyLine)
		t.app.QueueUpdateDraw(func() {
			if err != nil {
				// Store error for printing at exit
				t.storeErrorf("[Unload SSH Key] Failed to unload key '%s': %v", keyName, err)
				t.showStatusTempColor(fmt.Sprintf("Failed to unload key: %v", err), "#FF6B6B")
			} else {
				t.refreshSSHKeysList()
				t.showStatusTemp(fmt.Sprintf("Key %s unloaded successfully", keyName))
			}
		})
	}(key.PublicKeyLine, key.Name)
}

// handleLoadServerKey loads the SSH key associated with the selected server.
func (t *tui) handleLoadServerKey() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get selected server from Servers panel
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		t.showStatusTempColor("No server selected", "#FF6B6B")
		return
	}

	// Get the server's SSH key
	sshKey := t.getSSHKeyForServer(server)
	if sshKey == nil {
		t.showStatusTempColor("No SSH key found for this server", "#FF6B6B")
		return
	}

	if sshKey.LoadedInAgent {
		t.showStatusTempColor("Key already loaded in ssh-agent", "#FFA500")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Loading key %s...", sshKey.Name))

	// LoadKeyToAgent needs user interaction for passphrase
	// Suspend is blocking, so we can do this synchronously
	keyPath := sshKey.Path
	keyName := sshKey.Name

	t.app.Suspend(func() {
		err := t.gitService.LoadKeyToAgent(keyPath)
		if err != nil {
			// Store error for printing at exit
			t.storeErrorf("[Load Server SSH Key] Failed to load key '%s' (path: %s): %v", keyName, keyPath, err)
			fmt.Printf("\nFailed to load key: %v\n", err)
			fmt.Print("Press Enter to continue...")
			var dummy string
			_, _ = fmt.Scanln(&dummy)
		}
	})

	// Refresh UI after load (back in main thread)
	t.refreshServerList() // Refresh to update key status in server details
	t.refreshSSHKeysList()
	t.showStatusTemp(fmt.Sprintf("Key %s loaded successfully", keyName))
}

// handleUnloadServerKey unloads the SSH key associated with the selected server.
func (t *tui) handleUnloadServerKey() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get selected server from Servers panel
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		t.showStatusTempColor("No server selected", "#FF6B6B")
		return
	}

	// Get the server's SSH key
	sshKey := t.getSSHKeyForServer(server)
	if sshKey == nil {
		t.showStatusTempColor("No SSH key found for this server", "#FF6B6B")
		return
	}

	if !sshKey.LoadedInAgent {
		t.showStatusTempColor("Key not loaded in ssh-agent", "#FFA500")
		return
	}

	if sshKey.PublicKeyLine == "" {
		t.showStatusTempColor("Key has no public key line, cannot unload", "#FF6B6B")
		return
	}

	t.showStatusTemp(fmt.Sprintf("Unloading key %s...", sshKey.Name))

	go func(publicKeyLine string, keyName string) {
		err := t.gitService.UnloadKeyFromAgent(publicKeyLine)
		t.app.QueueUpdateDraw(func() {
			if err != nil {
				// Store error for printing at exit
				t.storeErrorf("[Unload Server SSH Key] Failed to unload key '%s': %v", keyName, err)
				t.showStatusTempColor(fmt.Sprintf("Failed to unload key: %v", err), "#FF6B6B")
			} else {
				t.refreshServerList() // Refresh to update key status in server details
				t.refreshSSHKeysList()
				t.showStatusTemp(fmt.Sprintf("Key %s unloaded successfully", keyName))
			}
		})
	}(sshKey.PublicKeyLine, sshKey.Name)
}

// getSSHKeyForServer gets the SSH key associated with a server.
func (t *tui) getSSHKeyForServer(server domain.Server) *domain.SSHKey {
	if t.gitService == nil || t.serverRepo == nil {
		return nil
	}

	// Get the first identity file if available
	if len(server.IdentityFiles) == 0 {
		return nil
	}

	identityFile := server.IdentityFiles[0]

	// Expand ~ to home directory for comparison
	if strings.HasPrefix(identityFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			identityFile = filepath.Join(homeDir, identityFile[2:])
		}
	} else if identityFile == "~" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			identityFile = homeDir
		}
	}

	// Fetch all SSH keys and find the one matching this identity file
	allKeys, err := t.gitService.ListAllSSHKeys(t.serverRepo)
	if err != nil {
		return nil
	}

	for _, key := range allKeys {
		if key.Path == identityFile {
			return &key
		}
	}

	return nil
}

func (t *tui) handleGitSSHSetup() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	setup := NewGitSSHSetup(t.app, t.gitService, t.serverRepo).
		OnDone(t.handleModalClose).
		OnCancel(t.handleModalClose)

	if err := setup.Show(); err != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Cannot setup Git SSH key:\n\n%v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				t.handleModalClose()
			})
		t.app.SetRoot(modal, true)
	}
}

// handleEditKeyComment opens the modal for editing SSH key comments.
func (t *tui) handleEditKeyComment() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get the currently displayed key from SSHKeyDetails
	key := t.sshKeyDetails.GetCurrentKey()
	if key == nil {
		t.showStatusTempColor("No SSH key selected", "#FF6B6B")
		return
	}

	// Only allow editing keys with a known file path
	if key.Path == "" {
		t.showStatusTempColor("Cannot edit comment for keys without a file path", "#FFA500")
		return
	}

	editor := NewEditKeyComment(t.app).
		SetKey(key.Name, key.Path, key.Comment).
		OnSave(func(newComment string) {
			// For encrypted keys, suspend TUI to allow passphrase input
			if key.IsEncrypted {
				t.app.Suspend(func() {
					if err := t.gitService.UpdateKeyComment(key.Path, newComment); err != nil {
						fmt.Printf("\nFailed to update comment: %v\n", err)
						fmt.Print("Press Enter to continue...")
						var dummy string
						_, _ = fmt.Scanln(&dummy)
					}
				})
				// Refresh after suspend (back in main thread)
				t.refreshSSHKeysList()
				t.showStatusTemp("SSH key comment updated successfully")
			} else {
				// For unencrypted keys, update directly
				if err := t.gitService.UpdateKeyComment(key.Path, newComment); err != nil {
					t.storeErrorf("[Edit SSH Key Comment] Failed to update comment for '%s': %v", key.Name, err)
					t.showStatusTempColor(fmt.Sprintf("Failed to update comment: %v", err), "#FF6B6B")
				} else {
					t.showStatusTemp("SSH key comment updated successfully")
					// Refresh the SSH keys list to show the updated comment
					t.refreshSSHKeysList()
				}
			}
		}).
		OnCancel(t.handleModalClose)

	if err := editor.Show(); err != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Cannot edit SSH key comment:\n\n%v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				t.handleModalClose()
			})
		t.app.SetRoot(modal, true)
	}
}

// handleEditServerKeyComment opens the modal for editing the SSH key comment of the selected server.
func (t *tui) handleEditServerKeyComment() {
	if t.gitService == nil {
		t.showStatusTempColor("Git service not available", "#FF6B6B")
		return
	}

	// Get the selected server
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		t.showStatusTempColor("No server selected", "#FF6B6B")
		return
	}

	// Get the SSH key for this server
	sshKey := t.getSSHKeyForServer(server)
	if sshKey == nil {
		t.showStatusTempColor("No SSH key found for this server", "#FF6B6B")
		return
	}

	// Only allow editing keys with a known file path
	if sshKey.Path == "" {
		t.showStatusTempColor("Cannot edit comment for keys without a file path", "#FFA500")
		return
	}

	// Remember if the key was loaded before editing
	wasLoaded := sshKey.LoadedInAgent

	editor := NewEditKeyComment(t.app).
		SetKey(sshKey.Name, sshKey.Path, sshKey.Comment).
		OnSave(func(newComment string) {
			// For encrypted keys, suspend TUI to allow passphrase input
			if sshKey.IsEncrypted {
				t.app.Suspend(func() {
					if err := t.gitService.UpdateKeyComment(sshKey.Path, newComment); err != nil {
						fmt.Printf("\nFailed to update comment: %v\n", err)
						fmt.Print("Press Enter to continue...")
						var dummy string
						_, _ = fmt.Scanln(&dummy)
					}
				})
			} else {
				// For unencrypted keys, update directly
				if err := t.gitService.UpdateKeyComment(sshKey.Path, newComment); err != nil {
					t.storeErrorf("[Edit SSH Key Comment] Failed to update comment for '%s': %v", sshKey.Name, err)
					t.showStatusTempColor(fmt.Sprintf("Failed to update comment: %v", err), "#FF6B6B")
					return
				}
			}

			// If the key was loaded in the agent, we need to unload and reload it
			// to get the fresh comment
			if wasLoaded {
				// Unload the key from agent
				if sshKey.PublicKeyLine != "" {
					if err := t.gitService.UnloadKeyFromAgent(sshKey.PublicKeyLine); err != nil {
						t.storeErrorf("[Edit SSH Key Comment] Failed to unload key: %v", err)
					}
				}

				t.showStatusTemp("SSH key comment updated! Press 'l' to reload the key in ssh-agent.")
			} else {
				t.showStatusTemp("SSH key comment updated successfully")
			}

			// Refresh the server details to show the updated comment
			if server, ok := t.serverList.GetSelectedServer(); ok {
				t.serverDetails.UpdateServer(server)
			}
			// Also refresh SSH keys list
			t.refreshSSHKeysList()
		}).
		OnCancel(t.handleModalClose)

	if err := editor.Show(); err != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Cannot edit SSH key comment:\n\n%v", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				t.handleModalClose()
			})
		t.app.SetRoot(modal, true)
	}
}

func (t *tui) refreshServerList() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)

	sortServersForUI(filtered, t.sortMode)
	t.serverList.UpdateServers(filtered)
}

func (t *tui) refreshSSHKeysList() {
	if t.gitService == nil {
		return
	}

	// Refresh agent keys
	agentKeys, err := t.gitService.ListSSHKeysFromAgent()
	if err != nil {
		t.showStatusTempColor(fmt.Sprintf("Failed to refresh SSH agent keys: %v", err), "#FF6B6B")
	} else {
		t.sshAgentKeysList.UpdateKeys(agentKeys)
	}

	// Update details for current selection if on SSH Agent panel
	if t.currentPanel == 1 {
		if key, ok := t.sshAgentKeysList.GetSelectedKey(); ok {
			t.sshKeyDetails.UpdateKey(key)
		}
	}
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}
