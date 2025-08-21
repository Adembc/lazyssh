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
	"github.com/mattn/go-runewidth"
)

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ",")
}

// renderTagBadgesForList renders up to two colored tag chips for the server list.
// If there are more tags, it appends a subtle gray "+N" badge. Returns an empty
// string when there are no tags to avoid cluttering the list.
func renderTagBadgesForList(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	max := 2
	shown := tags
	if len(tags) > max {
		shown = tags[:max]
	}
	parts := make([]string, 0, len(shown)+1)
	for _, t := range shown {
		// Light blue background chip, similar to details view.
		parts = append(parts, fmt.Sprintf("[black:#5FAFFF] %s [-:-:-]", t))
	}
	if extra := len(tags) - len(shown); extra > 0 {
		parts = append(parts, fmt.Sprintf("[#8A8A8A]+%d[-]", extra))
	}
	return strings.Join(parts, " ")
}

// cellPad pads a string with spaces so its display width is at least `width` cells.
// This keeps emoji-based icons from breaking alignment in tview.
func cellPad(s string, width int) string {
	w := runewidth.StringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func pinnedIcon(pinnedAt time.Time) string {
	// Use emojis for a nicer UI; combined with cellPad to keep widths consistent in tview.
	if pinnedAt.IsZero() {
		return "📡" // not pinned
	}
	return "📌" // pinned
}

func formatServerLine(s domain.Server) (primary, secondary string) {
	icon := cellPad(pinnedIcon(s.PinnedAt), 2)
	// Use a consistent color for alias; the icon reflects pinning
	primary = fmt.Sprintf("%s [white::b]%-12s[-] [#AAAAAA]%-18s[-] [#888888]Last SSH: %s[-]  %s", icon, s.Alias, s.Host, humanizeDuration(s.LastSeen), renderTagBadgesForList(s.Tags))
	secondary = ""
	return
}

func humanizeDuration(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm ago", h, m)
	}
	return fmt.Sprintf("%dm ago", m)
}

// BuildSSHCommand constructs a ready-to-run ssh command for the given server.
// Format: ssh [user@]host [-p PORT if not 22] [-i KEY if provided]
func BuildSSHCommand(s domain.Server) string {
	parts := []string{"ssh"}
	userHost := ""
	if s.User != "" && s.Host != "" {
		userHost = fmt.Sprintf("%s@%s", s.User, s.Host)
	} else if s.Host != "" {
		userHost = s.Host
	} else {
		userHost = s.Alias
	}
	parts = append(parts, userHost)

	if s.Port != 0 && s.Port != 22 {
		parts = append(parts, "-p", fmt.Sprintf("%d", s.Port))
	}
	if s.Key != "" {
		parts = append(parts, "-i", quoteIfNeeded(s.Key))
	}
	return strings.Join(parts, " ")
}

// quoteIfNeeded returns the value quoted if it contains spaces.
func quoteIfNeeded(val string) string {
	if strings.ContainsAny(val, " \t") {
		return fmt.Sprintf("\"%s\"", val)
	}
	return val
}
