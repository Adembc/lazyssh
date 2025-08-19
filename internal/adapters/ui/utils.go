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
)

func statusIcon(s string) string {
	switch s {
	case "online":
		return "🟢"
	case "warn":
		return "🟡"
	case "offline":
		return "🔴"
	default:
		return "⚪"
	}
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ",")
}

func formatServerLine(s domain.Server) (primary, secondary string) {
	icon := statusIcon(s.Status)
	// Choose a color per status for the alias and a subtle gray for host/time
	statusColor := "white"
	switch s.Status {
	case "online":
		statusColor = "green"
	case "warn":
		statusColor = "yellow"
	case "offline":
		statusColor = "red"
	}
	primary = fmt.Sprintf("%s [%s::b]%-12s[-] [#AAAAAA]%-18s[-] [#888888]Last SSH: %s[-]", icon, statusColor, s.Alias, s.Host, humanizeDuration(s.LastSeen))
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
