package ui

import (
	"fmt"

	"github.com/Adembc/lazyssh/internal/core/domain"
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
		t.app.Stop()
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
	case '?':
		t.handleHelpShow()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		t.handleServerConnect()
		return nil
	}

	return event
}

func (t *tui) handleSearchInput(query string) {
	filtered, _ := t.serverService.ListServers(query)
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
		t.showConnectModal(server)
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
	if original != nil {
		// Edit mode
		_ = t.serverService.UpdateServer(*original, server)
	} else {
		// Add mode
		_ = t.serverService.AddServer(server)
	}

	t.refreshServerList()
	t.handleFormCancel()
}

func (t *tui) handleServerDelete() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		_ = t.serverService.DeleteServer(server)
		t.refreshServerList()
	}
}

func (t *tui) handleFormCancel() {
	t.returnToMain()
}

func (t *tui) handleHelpShow() {
	t.showHelpModal()
}

func (t *tui) handleModalClose() {
	t.returnToMain()
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

func (t *tui) showConnectModal(server domain.Server) {
	msg := fmt.Sprintf("SSH to %s (%s@%s:%d)\n\nThis is a mock action.",
		server.Alias, server.User, server.Host, server.Port)

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.handleModalClose()
		})

	t.app.SetRoot(modal, true)
}

func (t *tui) showHelpModal() {
	text := "Keyboard shortcuts:\n\n" +
		"  ↑/↓          Navigate\n" +
		"  Enter        SSH connect (mock)\n" +
		"  a            Add server (mock)\n" +
		"  e            Edit server (mock)\n" +
		"  d            Delete entry (mock)\n" +
		"  /            Focus search\n" +
		"  q            Quit\n" +
		"  ?            Help\n"

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			t.handleModalClose()
		})

	t.app.SetRoot(modal, true)
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
		query = t.searchBar.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)
	t.serverList.UpdateServers(filtered)
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}
