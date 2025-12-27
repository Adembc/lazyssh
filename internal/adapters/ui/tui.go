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
	"sync"

	"github.com/gdamore/tcell/v2"
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
	serverRepo    ports.ServerRepository
	gitService    ports.GitService

	header           *AppHeader
	searchBar        *SearchBar
	serverList       *ServerList
	sshAgentKeysList *SSHKeysList // SSH keys from ssh-agent
	serverDetails    *ServerDetails
	sshKeyDetails    *SSHKeyDetails
	gitInfo          *GitInfo
	statusBar        *tview.TextView

	root    *tview.Flex
	left    *tview.Flex
	content *tview.Flex

	sortMode     SortMode
	currentPanel int // 0=Servers, 1=SSH Agent

	// Error storage for printing at exit
	errors      []string
	errorsMutex sync.Mutex
}

func NewTUI(logger *zap.SugaredLogger, ss ports.ServerService, sr ports.ServerRepository, gs ports.GitService, version, commit string) App {
	return &tui{
		logger:        logger,
		app:           tview.NewApplication(),
		serverService: ss,
		serverRepo:    sr,
		gitService:    gs,
		version:       version,
		commit:        commit,
	}
}

func (t *tui) Run() error {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Errorw("panic recovered", "error", r)
		}
		// Print any stored errors at exit for debugging
		t.printErrorsOnExit()
	}()
	t.app.EnableMouse(true)
	t.initializeTheme().buildComponents().buildLayout().bindEvents().loadInitialData()
	t.app.SetRoot(t.root, true)
	t.logger.Infow("starting TUI application", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.logger.Errorw("application run error", "error", err)
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
		OnEscape(t.blurSearchBar).
		OnNavigate(t.handleSearchNavigate)
	IsForwarding = t.serverService.IsForwarding

	t.serverList = NewServerList().
		OnSelectionChange(t.handleServerSelectionChange).
		OnReturnToSearch(t.handleReturnToSearch)

	t.sshAgentKeysList = NewSSHKeysList().
		OnSelectionChange(t.handleSSHKeySelectionChange)
	t.sshAgentKeysList.SetTitle(" SSH Agent ")

	// Add mouse handlers for panel selection
	t.serverList.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			t.handlePanelClick(0) // 0 = Servers panel
		}
		return action, event
	})
	t.sshAgentKeysList.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			t.handlePanelClick(1) // 1 = SSH Agent panel
		}
		return action, event
	})
	t.serverDetails = NewServerDetails(t.gitService, t.serverRepo)
	t.sshKeyDetails = NewSSHKeyDetails()
	t.gitInfo = NewGitInfo(t.gitService)
	t.statusBar = NewStatusBar()

	// default sort mode
	t.sortMode = SortByAliasAsc

	return t
}

func (t *tui) buildLayout() *tui {
	t.updateLeftPanel()

	// Right panel shows details based on focus
	right := tview.NewFlex().SetDirection(tview.FlexRow)
	if t.currentPanel != 0 {
		right.AddItem(t.sshKeyDetails, 0, 1, false)
	} else {
		right.AddItem(t.serverDetails, 0, 1, false)
	}

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(t.left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 2, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
	return t
}

func (t *tui) updateLeftPanel() {
	t.gitInfo.Update()

	// Rebuild left panel: SearchBar, Servers, SSH Agent, Git Info
	t.left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.searchBar, 3, 0, false).
		AddItem(t.serverList, 0, 1, true).
		AddItem(t.sshAgentKeysList, 0, 1, false)

	if t.gitInfo.IsVisible() {
		t.left.AddItem(t.gitInfo, 4, 0, false)
	}
}

func (t *tui) updateGitInfoPanel() {
	// Alias for updateLeftPanel to maintain compatibility
	t.updateLeftPanel()
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

	// Load SSH keys from ssh-agent
	if t.gitService != nil {
		agentKeys, err := t.gitService.ListSSHKeysFromAgent()
		if err == nil {
			t.sshAgentKeysList.UpdateKeys(agentKeys)
		}
	}

	// Set initial border colors (Servers panel is focused by default)
	t.updatePanelBorders()

	return t
}

func (t *tui) updateListTitle() {
	if t.serverList != nil {
		t.serverList.SetTitle(" Servers — Sort: " + t.sortMode.String() + " ")
	}
}

func (t *tui) updatePanelBorders() {
	// Update border colors to show which panel is active
	dimColor := tcell.Color238
	activeColor := tcell.ColorBlue

	// Selected background colors
	dimSelectionColor := tcell.Color238   // Gray for unfocused panel
	activeSelectionColor := tcell.Color24 // Blue for focused panel

	// Reset all borders and selections to dim
	t.serverList.SetBorderColor(dimColor)
	t.sshAgentKeysList.SetBorderColor(dimColor)
	t.serverList.SetSelectedBackgroundColor(dimSelectionColor)
	t.sshAgentKeysList.SetSelectedBackgroundColor(dimSelectionColor)

	// Highlight the active panel
	switch t.currentPanel {
	case 0:
		t.serverList.SetBorderColor(activeColor)
		t.serverList.SetSelectedBackgroundColor(activeSelectionColor)
	case 1:
		t.sshAgentKeysList.SetBorderColor(activeColor)
		t.sshAgentKeysList.SetSelectedBackgroundColor(activeSelectionColor)
	}
}

func (t *tui) updateRightPanel() {
	// Rebuild the right panel to show appropriate details
	if t.content == nil {
		return
	}

	right := tview.NewFlex().SetDirection(tview.FlexRow)
	if t.currentPanel == 0 {
		// Servers panel - show server details
		right.AddItem(t.serverDetails, 0, 1, false)
	} else {
		// SSH Keys or SSH Agent panel - show key details
		right.AddItem(t.sshKeyDetails, 0, 1, false)
	}

	// Rebuild content with updated right panel
	t.content.Clear()
	t.content.AddItem(t.left, 0, 3, true)
	t.content.AddItem(right, 0, 2, false)
}

// storeErrorf stores an error for later printing on exit.
// Keeps only the last 10 errors in memory.
func (t *tui) storeErrorf(format string, args ...interface{}) {
	t.errorsMutex.Lock()
	defer t.errorsMutex.Unlock()
	msg := fmt.Sprintf(format, args...)
	t.errors = append(t.errors, msg)
	// Keep only the last 10 errors
	if len(t.errors) > 10 {
		t.errors = t.errors[len(t.errors)-10:]
	}
}

// printErrorsOnExit prints any stored errors to stdout when the app exits.
func (t *tui) printErrorsOnExit() {
	t.errorsMutex.Lock()
	errors := make([]string, len(t.errors))
	copy(errors, t.errors)
	t.errorsMutex.Unlock()

	if len(errors) > 0 {
		fmt.Println("\n=== Errors during session ===")
		for _, err := range errors {
			fmt.Println(err)
		}
		fmt.Println("==============================")
	}
}
