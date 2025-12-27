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

	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/Adembc/lazyssh/internal/core/services"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GitSSHSetup struct {
	*tview.Flex
	app        *tview.Application
	gitService ports.GitService
	serverRepo ports.ServerRepository
	form       *tview.Form
	infoText   *tview.TextView
	onDone     func()
	onCancel   func()

	selectedKey   string
	selectedScope string
	keys          []ports.SSHKey
	repoPath      string
}

func NewGitSSHSetup(app *tview.Application, gitService ports.GitService, serverRepo ports.ServerRepository) *GitSSHSetup {
	setup := &GitSSHSetup{
		Flex:       tview.NewFlex().SetDirection(tview.FlexRow),
		app:        app,
		gitService: gitService,
		serverRepo: serverRepo,
		form:       tview.NewForm(),
		infoText:   tview.NewTextView(),
	}

	setup.infoText.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 2, 2)

	setup.form.SetBorderPadding(1, 1, 2, 2)

	setup.AddItem(setup.infoText, 0, 1, false).
		AddItem(setup.form, 0, 2, true)

	setup.SetBorder(true).
		SetTitle(" Configure Git SSH Key ").
		SetTitleAlign(tview.AlignLeft)

	return setup
}

func (g *GitSSHSetup) OnDone(handler func()) *GitSSHSetup {
	g.onDone = handler
	return g
}

func (g *GitSSHSetup) OnCancel(handler func()) *GitSSHSetup {
	g.onCancel = handler
	return g
}

func (g *GitSSHSetup) Show() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a git repository
	if !g.gitService.IsGitRepository(cwd) {
		return fmt.Errorf("not in a git repository")
	}

	// Get the git root path
	repoPath, err := g.gitService.GetGitRootPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to get git root path: %w", err)
	}
	g.repoPath = repoPath

	// Get current SSH config
	currentConfig, _ := g.gitService.GetCurrentGitSSHConfig(repoPath)

	// List available SSH keys
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	sshDir := filepath.Join(homeDir, ".ssh")

	keys, err := g.gitService.ListSSHKeys(sshDir, g.serverRepo)
	if err != nil {
		return fmt.Errorf("failed to list SSH keys: %w", err)
	}
	g.keys = keys

	if len(keys) == 0 {
		return fmt.Errorf("no SSH keys found in %s", sshDir)
	}

	// Build info text
	info := fmt.Sprintf("[yellow]Git Repository:[-] %s\n\n", repoPath)
	if currentConfig != "" {
		info += fmt.Sprintf("[green]Current SSH Config:[-]\n%s\n\n", currentConfig)
	} else {
		info += "[dim]No SSH key configured for Git[-]\n\n"
	}

	// Count keys in agent
	keysInAgent := 0
	for _, key := range keys {
		if key.InAgent {
			keysInAgent++
		}
	}
	if keysInAgent > 0 {
		info += fmt.Sprintf("[green]%d key(s) loaded in ssh-agent/keychain[-]\n\n", keysInAgent)
	}

	info += "Select an SSH key to use for Git operations in this repository.\n"
	info += "The configuration will be saved using [yellow]git config core.sshCommand[-]."

	g.infoText.SetText(info)

	// Build form
	g.form.Clear(true)

	// Add key dropdown
	keyOptions := make([]string, len(keys))
	for i, key := range keys {
		status := ""
		switch {
		case key.InAgent:
			status = " [green](in agent)[-]"
		case !key.HasPubKey:
			status = " [red](no .pub)[-]"
		case key.IsEncrypted:
			status = " [yellow](encrypted)[-]"
		}
		keyOptions[i] = fmt.Sprintf("%s%s", key.Name, status)
	}

	g.form.AddDropDown("SSH Key", keyOptions, 0, func(option string, optionIndex int) {
		if optionIndex >= 0 && optionIndex < len(g.keys) {
			g.selectedKey = g.keys[optionIndex].Path
		}
	})

	// Add scope dropdown
	scopeOptions := []string{"local (this repository only)", "global (all repositories)"}
	g.form.AddDropDown("Scope", scopeOptions, 0, func(option string, optionIndex int) {
		if optionIndex == 0 {
			g.selectedScope = services.ScopeLocal
		} else {
			g.selectedScope = services.ScopeGlobal
		}
	})

	// Set initial values
	if len(g.keys) > 0 {
		g.selectedKey = g.keys[0].Path
	}
	g.selectedScope = services.ScopeLocal

	// Add buttons
	g.form.AddButton("Configure", func() {
		if err := g.gitService.ConfigureGitSSHKey(g.repoPath, g.selectedKey, g.selectedScope); err != nil {
			// Show error modal
			errorModal := tview.NewModal().
				SetText(fmt.Sprintf("Configuration failed:\n%v", err)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					g.app.SetRoot(g, true)
					g.app.SetFocus(g.form)
				})
			g.app.SetRoot(errorModal, true)
			return
		}

		// Show success modal
		successModal := tview.NewModal().
			SetText(fmt.Sprintf("Git configured successfully!\n\nSSH Key: %s\nScope: %s",
				filepath.Base(g.selectedKey), g.selectedScope)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if g.onDone != nil {
					g.onDone()
				}
			})
		g.app.SetRoot(successModal, true)
	})

	// Add "Clear Configuration" button if there's an existing config
	if currentConfig != "" {
		g.form.AddButton("Clear Configuration", func() {
			// Show confirmation modal
			confirmModal := tview.NewModal().
				SetText("Clear Git SSH configuration?\n\nThis will reset Git to use default SSH behavior.").
				AddButtons([]string{"Cancel", "Clear Local", "Clear Both"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonIndex == 0 {
						// Cancel - return to setup form
						g.app.SetRoot(g, true)
						g.app.SetFocus(g.form)
						return
					}

					scope := services.ScopeLocal
					if buttonIndex == 2 {
						scope = services.ScopeBoth
					}

					// Clear the configuration
					if err := g.gitService.ClearGitSSHConfig(g.repoPath, scope); err != nil {
						errorModal := tview.NewModal().
							SetText(fmt.Sprintf("Failed to clear configuration:\n%v", err)).
							AddButtons([]string{"OK"}).
							SetDoneFunc(func(buttonIndex int, buttonLabel string) {
								g.app.SetRoot(g, true)
								g.app.SetFocus(g.form)
							})
						g.app.SetRoot(errorModal, true)
						return
					}

					// Show success modal
					scopeText := "local"
					if scope == services.ScopeBoth {
						scopeText = "local and global"
					}
					successModal := tview.NewModal().
						SetText(fmt.Sprintf("Git SSH configuration cleared!\n\nScope: %s\n\nGit will now use default SSH behavior.", scopeText)).
						AddButtons([]string{"OK"}).
						SetDoneFunc(func(buttonIndex int, buttonLabel string) {
							if g.onDone != nil {
								g.onDone()
							}
						})
					g.app.SetRoot(successModal, true)
				})
			g.app.SetRoot(confirmModal, true)
		})
	}

	g.form.AddButton("Cancel", func() {
		if g.onCancel != nil {
			g.onCancel()
		}
	})

	// Handle keyboard shortcuts
	g.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if g.onCancel != nil {
				g.onCancel()
			}
			return nil
		}
		return event
	})

	g.app.SetRoot(g, true)
	g.app.SetFocus(g.form)

	return nil
}
