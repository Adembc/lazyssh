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
	"go.uber.org/zap"

	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/rivo/tview"
)

type App interface {
	Run() error
}

type tui struct {
	logger *zap.SugaredLogger

	version string
	commit  string

	app           *tview.Application
	serverService ports.ServerService

	header     *AppHeader
	searchBar  *SearchBar
	serverList *ServerList
	details    *ServerDetails
	statusBar  *tview.TextView

	root    *tview.Flex
	left    *tview.Flex
	content *tview.Flex

	sortMode     SortMode
	themeWatcher *ThemeWatcher
}

func NewTUI(logger *zap.SugaredLogger, ss ports.ServerService, version, commit string) App {
	return &tui{
		logger:        logger,
		app:           tview.NewApplication(),
		serverService: ss,
		version:       version,
		commit:        commit,
	}
}

func (t *tui) Run() error {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Errorw("panic recovered", "error", r)
		}
	}()
	t.app.EnableMouse(true)
	t.initializeTheme()
	t.initializeThemeWatcher()
	t.buildComponents()
	t.buildLayout()
	t.bindEvents()
	t.loadInitialData()
	t.app.SetRoot(t.root, true)
	t.logger.Infow("starting TUI application", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.logger.Errorw("application run error", "error", err)
		return err
	}
	t.stopThemeWatcher()
	return nil
}

func (t *tui) initializeTheme() {
	ApplyTheme()
}

func (t *tui) initializeThemeWatcher() {
	t.themeWatcher = NewThemeWatcher(func(newTheme string) {
		// Only react if we're in system theme mode
		if CurrentThemeMode != ThemeSystem {
			return
		}
		// Check if the theme actually changed
		if newTheme == CurrentTheme.Name {
			return
		}
		// Apply the new theme on the UI thread
		t.app.QueueUpdateDraw(func() {
			if newTheme == ThemeLight {
				CurrentTheme = &LightTheme
			} else {
				CurrentTheme = &DarkTheme
			}
			ApplyTheme()
			t.rebuildUI()
			t.showStatusTemp("Theme: " + newTheme + " (system)")
		})
	})
	// Start watching if system theme is selected
	if CurrentThemeMode == ThemeSystem {
		t.themeWatcher.Start()
	}
}

func (t *tui) stopThemeWatcher() {
	if t.themeWatcher != nil {
		t.themeWatcher.Stop()
	}
}

func (t *tui) buildComponents() {
	t.header = NewAppHeader(t.version, t.commit, RepoURL)
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.blurSearchBar).
		OnNavigate(t.handleSearchNavigate)
	IsForwarding = t.serverService.IsForwarding

	t.serverList = NewServerList().
		OnSelectionChange(t.handleServerSelectionChange).
		OnReturnToSearch(t.handleReturnToSearch)
	t.details = NewServerDetails()
	t.statusBar = NewStatusBar()

	// default sort mode
	t.sortMode = SortByAliasAsc
}

func (t *tui) buildLayout() {
	t.left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.searchBar, 3, 0, false).
		AddItem(t.serverList, 0, 1, true)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.details, 0, 1, false)

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(t.left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 2, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
}

func (t *tui) bindEvents() {
	t.root.SetInputCapture(t.handleGlobalKeys)
}

func (t *tui) loadInitialData() {
	servers, _ := t.serverService.ListServers("")
	sortServersForUI(servers, t.sortMode)
	t.updateListTitle()
	t.serverList.UpdateServers(servers)
}

func (t *tui) updateListTitle() {
	if t.serverList != nil {
		t.serverList.SetTitle(" Servers — Sort: " + t.sortMode.String() + " ")
	}
}

// rebuildUI rebuilds all UI components to apply theme changes.
// It preserves the current state (search query, selection, sort mode).
func (t *tui) rebuildUI() {
	// Save current state
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	currentIdx := 0
	if t.serverList != nil {
		currentIdx = t.serverList.GetCurrentItem()
	}

	// Rebuild components
	t.buildComponents()
	t.buildLayout()
	t.bindEvents()

	// Restore state
	if query != "" {
		t.searchBar.InputField.SetText(query)
	}
	t.loadInitialData()
	if currentIdx >= 0 && currentIdx < t.serverList.GetItemCount() {
		t.serverList.SetCurrentItem(currentIdx)
	}
	if srv, ok := t.serverList.GetSelectedServer(); ok {
		t.details.UpdateServer(srv)
	}

	// Update display
	t.app.SetRoot(t.root, true)
	t.app.SetFocus(t.serverList)
}
