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
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/rivo/tview"
)

type tui struct {
	version string
	commit  string

	app           *tview.Application
	serverService ports.ServerService

	header     *AppHeader
	searchBar  *SearchBar
	hintBar    *tview.TextView
	serverList *ServerList
	details    *ServerDetails
	statusBar  *tview.TextView

	root    *tview.Flex
	left    *tview.Flex
	content *tview.Flex

	sortMode      SortMode
	searchVisible bool
}

func NewTUI(ss ports.ServerService, version, commit string) *tui {
	return &tui{
		app:           tview.NewApplication(),
		serverService: ss,
		version:       version,
		commit:        commit,
	}
}

func (t *tui) Run() error {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic recovered", "error", r)
		}
	}()
	t.app.EnableMouse(true)
	t.initializeTheme().buildComponents().buildLayout().bindEvents().loadInitialData().loadSplashScreen()
	slog.Info("starting TUI application", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		slog.Error("application run error", "error", err)
		return err
	}
	return nil
}

func (t *tui) initializeTheme() *tui {
	tview.Styles.PrimitiveBackgroundColor = tcell.Color232
	tview.Styles.ContrastBackgroundColor = tcell.Color235
	tview.Styles.BorderColor = tcell.Color238
	tview.Styles.TitleColor = tcell.Color250
	tview.Styles.PrimaryTextColor = tcell.Color252
	tview.Styles.TertiaryTextColor = tcell.Color245
	tview.Styles.SecondaryTextColor = tcell.Color245
	tview.Styles.GraphicsColor = tcell.Color238
	return t
}

func (t *tui) buildComponents() *tui {
	t.header = NewAppHeader(t.version, t.commit, RepoURL)
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.hideSearchBar)
	t.hintBar = NewHintBar()
	t.serverList = NewServerList().
		OnSelectionChange(t.handleServerSelectionChange)
	t.details = NewServerDetails()
	t.statusBar = NewStatusBar()

	// default sort mode
	t.sortMode = SortByAliasAsc

	return t
}

func (t *tui) buildLayout() *tui {
	t.left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.hintBar, 1, 0, false).
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
	return t
}

func (t *tui) bindEvents() *tui {
	t.root.SetInputCapture(t.handleGlobalKeys)
	return t
}

func (t *tui) loadInitialData() *tui {
	servers, _ := t.serverService.ListServers("")
	sortServersForUI(servers, t.sortMode)
	t.updateListTitle()
	t.serverList.UpdateServers(servers)

	return t
}

func (t *tui) updateListTitle() {
	if t.serverList != nil {
		t.serverList.SetTitle("Servers — Sort: " + t.sortMode.String())
	}
}

func (t *tui) loadSplashScreen() *tui {
	splash, stop := buildSplash(t.app)
	t.app.SetRoot(splash, true)
	time.AfterFunc(SplashScreenDuration, func() {
		stop()
		t.app.QueueUpdateDraw(func() {
			t.app.SetRoot(t.root, true)
		})
	})
	return t
}
