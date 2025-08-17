package ui

import (
	"fmt"
	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerDetails struct {
	*tview.TextView
}

func NewServerDetails() *ServerDetails {
	details := &ServerDetails{
		TextView: tview.NewTextView(),
	}
	details.build()
	return details
}

func (sd *ServerDetails) build() {
	sd.SetDynamicColors(true).
		SetWrap(true).
		SetBorder(true).
		SetTitle("Details").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
}

func (sd *ServerDetails) UpdateServer(server domain.Server) {
	text := fmt.Sprintf(
		"[::b]%s[-]\n\nHost: [white]%s[-]\nUser: [white]%s[-]\nPort: [white]%d[-]\nKey:  [white]%s[-]\nTags: [white]%s[-]\nStatus: %s\nLast: %s\n\n[::b]Commands:[-]\n  Enter: SSH connect\n  a: Add new server\n  e: Edit entry\n  d: Delete entry",
		server.Alias, server.Host, server.User, server.Port,
		server.Key, joinTags(server.Tags), statusIcon(server.Status),
		server.LastSeen.Format("2006-01-02 15:04"))
	sd.SetText(text)
}

func (sd *ServerDetails) ShowEmpty() {
	sd.SetText("No servers match the current filter.")
}
