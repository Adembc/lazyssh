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
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppHeader struct {
	*tview.Flex
	version   string
	gitCommit string
	repoURL   string
}

func NewAppHeader(version, gitCommit, repoURL string) *AppHeader {
	header := &AppHeader{
		Flex:      tview.NewFlex(),
		version:   version,
		repoURL:   repoURL,
		gitCommit: gitCommit,
	}
	header.build()
	return header
}

func (h *AppHeader) build() {
	headerBg := tcell.Color234

	left := h.buildLeftSection(headerBg)
	center := h.buildCenterSection(headerBg)
	right := h.buildRightSection(headerBg)

	headerBar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 1, false).
		AddItem(center, 0, 1, false).
		AddItem(right, 0, 1, false)

	separator := h.createSeparator()

	h.Flex.SetDirection(tview.FlexRow).
		AddItem(headerBar, 1, 0, false).
		AddItem(separator, 1, 0, false)
}

func (h *AppHeader) buildLeftSection(bg tcell.Color) *tview.TextView {
	left := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	left.SetBackgroundColor(bg)
	stylizedName := "🚀 [#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
	left.SetText(stylizedName)
	return left
}

func (h *AppHeader) buildCenterSection(bg tcell.Color) *tview.TextView {
	center := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	center.SetBackgroundColor(bg)

	commit := shortCommit(h.gitCommit)

	// Build tag-like chips for version, commit, and build time
	versionTag := makeTag(h.version, "#22C55E") // green
	commitTag := ""
	if commit != "" {
		commitTag = makeTag(commit, "#A78BFA") // violet
	}

	text := versionTag
	if commitTag != "" {
		text += "  " + commitTag
	}

	center.SetText(text)
	return center
}

func (h *AppHeader) buildRightSection(bg tcell.Color) *tview.TextView {
	right := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	right.SetBackgroundColor(bg)
	currentTime := time.Now().Format("Mon, 02 Jan 2006 15:04")
	right.SetText("[#55AAFF::u]🔗 " + h.repoURL + "[-]  [#AAAAAA]• " + currentTime + "[-]")
	return right
}

func (h *AppHeader) createSeparator() *tview.TextView {
	separator := tview.NewTextView().SetDynamicColors(true)
	separator.SetBackgroundColor(tcell.Color235)
	separator.SetText("[#444444]" + strings.Repeat("─", 200) + "[-]")
	return separator
}

// shortCommit returns first 7 chars of commit if it looks valid; otherwise empty string.
func shortCommit(c string) string {
	c = strings.TrimSpace(c)
	if c == "" || c == "unknown" || c == "(devel)" {
		return ""
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// makeTag returns a rectangular-looking colored chip for the given text.
func makeTag(text, bg string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "[black:" + bg + "::b]  " + text + "  [-]"
}
