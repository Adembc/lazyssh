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

	"github.com/Adembc/lazyssh/internal/core/domain"
	tcell "github.com/gdamore/tcell/v2" // Explicit alias
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

func (sd *ServerDetails) UpdateServer(server domain.Server) {
	text := fmt.Sprintf(
		"[::b]%s[-]\n\nHost: [white]%s[-]\nUser: [white]%s[-]\nPort: [white]%d[-]\nKey:  [white]%s[-]\nTags: [white]%s[-]\nStatus: %s\nLast: %s\n\n[::b]Commands:[-]\n  Enter: SSH connect\n  a: Add new server\n  e: Edit entry\n  d: Delete entry",
		server.Alias, server.Host, server.User, server.Port,
		server.Key, joinTags(server.Tags), statusIcon(server.Status),
		server.LastSeen.Format("2006-01-02 15:04"))
	sd.TextView.SetText(text)
}

func (sd *ServerDetails) ShowEmpty() {
	sd.TextView.SetText("No servers match the current filter.")
}
