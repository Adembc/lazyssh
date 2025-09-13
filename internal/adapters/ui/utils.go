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

// renderTagBadgesForList renders up to two colored tag chips for the server list.
// If there are more tags, it appends a subtle gray "+N" badge. Returns an empty
// string when there are no tags to avoid cluttering the list.
func renderTagBadgesForList(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	maxTags := 2
	shown := tags
	if len(tags) > maxTags {
		shown = tags[:maxTags]
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

func formatServerLine(s domain.Server, width int) (primary, secondary string) {
	icon := cellPad(pinnedIcon(s.PinnedAt), 2)

	// Build main content
	mainText := fmt.Sprintf("%s [white::b]%-12s[-] [#AAAAAA]%-18s[-] [#888888]Last SSH: %s[-]  %s",
		icon, s.Alias, s.Host, humanizeDuration(s.LastSeen), renderTagBadgesForList(s.Tags))

	// Format ping status (4 chars max for value)
	pingIndicator := ""
	if s.PingStatus != "" {
		switch s.PingStatus {
		case "up":
			if s.PingLatency > 0 {
				ms := s.PingLatency.Milliseconds()
				var statusText string
				if ms < 100 {
					statusText = fmt.Sprintf("%dms", ms) // e.g., "57ms"
				} else {
					// Format as #.#s for >= 100ms
					seconds := float64(ms) / 1000.0
					statusText = fmt.Sprintf("%.1fs", seconds) // e.g., "0.3s", "1.5s"
				}
				// Ensure exactly 4 chars
				statusText = fmt.Sprintf("%-4s", statusText)
				pingIndicator = fmt.Sprintf("[#4AF626]● %s[-]", statusText)
			} else {
				pingIndicator = "[#4AF626]● UP  [-]"
			}
		case "down":
			pingIndicator = "[#FF6B6B]● DOWN[-]"
		case "checking":
			pingIndicator = "[#FFB86C]● ... [-]"
		}
	}

	// Calculate padding for right alignment
	if pingIndicator != "" && width > 0 {
		// Strip color codes to calculate real length
		mainTextLen := len(stripSimpleColors(mainText))
		indicatorLen := 6 // "● XXXX" is always 6 display chars

		// Calculate padding needed
		switch {
		case width > 80:
			// Wide screen: show full indicator
			paddingLen := width - mainTextLen - indicatorLen // No margin, stick to right edge
			if paddingLen < 1 {
				paddingLen = 1
			}
			padding := strings.Repeat(" ", paddingLen)
			primary = fmt.Sprintf("%s%s%s", mainText, padding, pingIndicator)
		case width > 60:
			// Medium screen: show only dot
			simplePingIndicator := ""
			switch s.PingStatus {
			case "up":
				simplePingIndicator = "[#4AF626]●[-]"
			case "down":
				simplePingIndicator = "[#FF6B6B]●[-]"
			case "checking":
				simplePingIndicator = "[#FFB86C]●[-]"
			}
			paddingLen := width - mainTextLen - 1 // 1 for dot, no margin
			if paddingLen < 1 {
				paddingLen = 1
			}
			padding := strings.Repeat(" ", paddingLen)
			primary = fmt.Sprintf("%s%s%s", mainText, padding, simplePingIndicator)
		default:
			// Narrow screen: no ping indicator
			primary = mainText
		}
	} else {
		primary = mainText
	}

	secondary = ""
	return primary, secondary
}

// stripSimpleColors removes basic tview color codes for length calculation
func stripSimpleColors(s string) string {
	result := s
	// Remove color tags like [#FFFFFF] or [-]
	for {
		start := strings.Index(result, "[")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "]")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}
	return result
}

func humanizeDuration(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	}
	if d < 48*time.Hour {
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	}
	if d < 60*24*time.Hour {
		days := int(d.Hours()) / 24
		return fmt.Sprintf("%dd ago", days)
	}
	if d < 365*24*time.Hour {
		months := int(d.Hours()) / (24 * 30)
		if months < 1 {
			months = 1
		}
		return fmt.Sprintf("%dmo ago", months)
	}
	years := int(d.Hours()) / (24 * 365)
	if years < 1 {
		years = 1
	}
	return fmt.Sprintf("%dy ago", years)
}

// BuildSSHCommand constructs a ready-to-run ssh command for the given server.
// Format: ssh [user@]host [-p PORT if not 22] [-i KEY if provided]
func BuildSSHCommand(s domain.Server) string {
	parts := []string{"ssh"}
	userHost := ""
	switch {
	case s.User != "" && s.Host != "":
		userHost = fmt.Sprintf("%s@%s", s.User, s.Host)
	case s.Host != "":
		userHost = s.Host
	default:
		userHost = s.Alias
	}
	parts = append(parts, userHost)

	if s.Port != 0 && s.Port != 22 {
		parts = append(parts, "-p", fmt.Sprintf("%d", s.Port))
	}
	if len(s.IdentityFiles) > 0 {
		parts = append(parts, "-i", quoteIfNeeded(s.IdentityFiles[0]))
	}
	return strings.Join(parts, " ")
}

// quoteIfNeeded returns the value quoted if it contains spaces.
func quoteIfNeeded(val string) string {
	if strings.ContainsAny(val, " \t") {
		return fmt.Sprintf("%q", val)
	}
	return val
}
