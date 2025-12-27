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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GitInfo struct {
	*tview.TextView
	gitService ports.GitService
	visible    bool
}

func NewGitInfo(gitService ports.GitService) *GitInfo {
	info := &GitInfo{
		TextView:   tview.NewTextView(),
		gitService: gitService,
		visible:    false,
	}

	info.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorder(true).
		SetTitle(" Git Repository ").
		SetTitleAlign(tview.AlignLeft).
		SetBackgroundColor(tcell.Color232).
		SetBorderColor(tcell.Color238)

	return info
}

func (g *GitInfo) Update() {
	if g.gitService == nil {
		g.visible = false
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		g.visible = false
		return
	}

	if !g.gitService.IsGitRepository(cwd) {
		g.visible = false
		return
	}

	repoPath, err := g.gitService.GetGitRootPath(cwd)
	if err != nil {
		g.visible = false
		return
	}

	g.visible = true

	// Get push remote URL with remote name
	remoteName, remoteURL, err := g.gitService.GetPushRemoteURL(repoPath)

	// Get current SSH config
	sshConfig, _ := g.gitService.GetCurrentGitSSHConfig(repoPath)

	// Build info text
	repoName := filepath.Base(repoPath)

	// First line: repo name with remote info
	if err == nil && remoteURL != "" {
		text := fmt.Sprintf("[yellow]%s[-] ([dim]%s:[-][blue]%s[-])", repoName, remoteName, remoteURL)
		g.SetText(text)
	} else {
		g.SetText(fmt.Sprintf("[yellow]%s[-]", repoName))
	}

	// Second line: SSH configured message
	if sshConfig != "" {
		g.SetText(g.GetText(false) + "\n[green]✓[-] SSH configured")
	} else {
		g.SetText(g.GetText(false) + "\n[dim]Press G to configure SSH[-]")
	}
}

func (g *GitInfo) IsVisible() bool {
	return g.visible
}
