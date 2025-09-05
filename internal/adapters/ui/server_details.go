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
	sd.TextView.SetDynamicColors(true).
		SetWrap(true).
		SetBorder(true).
		SetTitle("Details").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
}

// renderTagChips builds colored tag chips for details view.
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	chips := make([]string, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, fmt.Sprintf("[black:#5FAFFF] %s [-:-:-]", t))
	}
	return strings.Join(chips, " ")
}

func (sd *ServerDetails) UpdateServer(server domain.Server) {
	lastSeen := server.LastSeen.Format("2006-01-02 15:04:05")
	if server.LastSeen.IsZero() {
		lastSeen = "Never"
	}
	serverKey := server.Key
	if serverKey == "" && server.ConnectionType == domain.ConnectionTypeSSH {
		serverKey = "(default: ~/.ssh/id_{rsa,ed25519,ecdsa})"
	}
	pinnedStr := "true"
	if server.PinnedAt.IsZero() {
		pinnedStr = "false"
	}
	tagsText := renderTagChips(server.Tags)

	// Build base information
	var text strings.Builder
	text.WriteString(fmt.Sprintf("[::b]%s[-]\n\n", server.Alias))

	// Connection type and source
	connectionTypeStr := "SSH"
	if server.ConnectionType == domain.ConnectionTypeAWS {
		connectionTypeStr = "AWS SSM"
	}
	text.WriteString(fmt.Sprintf("Type: [white]%s[-]\nSource: [white]%s[-]\n", connectionTypeStr, server.Source))

	// Common fields
	text.WriteString(fmt.Sprintf("Host: [white]%s[-]\nUser: [white]%s[-]\n", server.Host, server.User))

	// Connection-specific fields
	if server.ConnectionType == domain.ConnectionTypeAWS {
		text.WriteString(fmt.Sprintf("AWS Profile: [white]%s[-]\n", server.AWSProfile))
		text.WriteString(fmt.Sprintf("AWS Region: [white]%s[-]\n", server.AWSRegion))
		text.WriteString(fmt.Sprintf("EC2 Filter: [white]%s[-]\n", server.EC2TagFilter))
		text.WriteString(fmt.Sprintf("SSM Document: [white]%s[-]\n", server.SSMDocument))
		if server.SSMCommand != "" {
			text.WriteString(fmt.Sprintf("SSM Command: [white]%s[-]\n", server.SSMCommand))
		}
	} else {
		text.WriteString(fmt.Sprintf("Port: [white]%d[-]\n", server.Port))
		text.WriteString(fmt.Sprintf("Key:  [white]%s[-]\n", serverKey))
	}

	// Common metadata
	text.WriteString(fmt.Sprintf("Tags: %s\nPinned: [white]%s[-]\nLast SSH: %s\nSSH Count: [white]%d[-]\n\n",
		tagsText, pinnedStr, lastSeen, server.SSHCount))

	// Commands (different for AWS vs SSH)
	text.WriteString("[::b]Commands:[-]\n")
	if server.ConnectionType == domain.ConnectionTypeAWS {
		text.WriteString("  Enter: AWS SSM connect\n  r: Refresh list\n  t: Edit tags\n  p: Pin/Unpin\n  (AWS servers are read-only)")
	} else {
		text.WriteString("  Enter: SSH connect\n  c: Copy SSH command\n  g: Ping server\n  r: Refresh list\n  a: Add new server\n  e: Edit entry\n  t: Edit tags\n  d: Delete entry\n  p: Pin/Unpin")
	}

	sd.TextView.SetText(text.String())
}

func (sd *ServerDetails) ShowEmpty() {
	sd.TextView.SetText("No servers match the current filter.")
}
