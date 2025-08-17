package ui

import (
	"fmt"
	"github.com/Adembc/lazyssh/internal/core/domain"
	"strings"
	"time"
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
	primary = fmt.Sprintf("%s [%s::b]%-12s[-] [#AAAAAA]%-18s[-] [#888888]Last:%s[-]", icon, statusColor, s.Alias, s.Host, humanizeDuration(time.Since(s.LastSeen)))
	secondary = ""
	return
}

func humanizeDuration(d time.Duration) string {
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
