package ui

import (
	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
)

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
		t.showSearchBar()
		return nil
	case 'a':
		t.showAddServerForm()
		return nil
	case 'e':
		t.showEditServerForm()
		return nil
	case 'd':
		t.deleteSelectedServer()
		return nil
	case '?':
		t.showHelpModal()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		t.connectToSelectedServer()
		return nil
	}

	return event
}

func (t *tui) handleSearch(query string) {
	filtered, _ := t.serverService.ListServers(query)
	t.serverList.UpdateServers(filtered)
	if len(filtered) == 0 {
		t.details.ShowEmpty()
	}
}

func (t *tui) handleServerConnect(server domain.Server) {
	showConnectModal(t.app, t.root, server)
}

func (t *tui) handleServerSelectionChange(server domain.Server) {
	t.details.UpdateServer(server)
}

func (t *tui) showSearchBar() {
	t.left.Clear()
	t.left.AddItem(t.searchBar, 3, 0, true)
	t.left.AddItem(t.serverList, 0, 1, false)
	t.app.SetFocus(t.searchBar)
	t.searchVisible = true
}

func (t *tui) hideSearchBar() {
	t.left.Clear()
	t.left.AddItem(t.hintBar, 1, 0, false)
	t.left.AddItem(t.serverList, 0, 1, true)
	t.app.SetFocus(t.serverList)
	t.searchVisible = false
}

func (t *tui) showAddServerForm() {
	form := NewServerForm(ServerFormAdd, nil).
		OnSave(t.handleServerSave).
		OnCancel(t.returnToMain)
	t.app.SetRoot(form, true)
}

func (t *tui) showEditServerForm() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		form := NewServerForm(ServerFormEdit, &server).
			OnSave(t.handleServerSave).
			OnCancel(t.returnToMain)
		t.app.SetRoot(form, true)
	}
}

func (t *tui) handleServerSave(server domain.Server, original *domain.Server) {
	if original != nil {
		// Edit mode
		t.serverService.UpdateServer(*original, server)
	} else {
		// Add mode
		t.serverService.AddServer(server)
	}

	t.refreshServerList()
	t.returnToMain()
}

func (t *tui) deleteSelectedServer() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		_ = t.serverService.DeleteServer(server)
		t.refreshServerList()
	}
}

func (t *tui) refreshServerList() {
	query := ""
	if t.searchVisible {
		query = t.searchBar.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)
	t.serverList.UpdateServers(filtered)
}

func (t *tui) connectToSelectedServer() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.handleServerConnect(server)
	}
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}

func (t *tui) showHelpModal() {
	showHelpModal(t.app, t.root)
}
