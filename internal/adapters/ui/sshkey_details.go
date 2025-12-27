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

type SSHKeyDetails struct {
	*tview.TextView
	currentKey *domain.SSHKey
}

func NewSSHKeyDetails() *SSHKeyDetails {
	details := &SSHKeyDetails{
		TextView: tview.NewTextView(),
	}
	details.build()
	return details
}

func (kd *SSHKeyDetails) build() {
	kd.TextView.SetDynamicColors(true).
		SetWrap(true).
		SetBorder(true).
		SetTitle(" SSH Key Details ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
}

func (kd *SSHKeyDetails) UpdateKey(key domain.SSHKey) {
	kd.currentKey = &key
	var details strings.Builder

	// Key name
	details.WriteString(fmt.Sprintf("[yellow]%s[-]\n\n", key.Name))

	// Basic info section
	details.WriteString("[white::b]Basic Info:[-]\n")
	details.WriteString(fmt.Sprintf("  [dim]Path:[-] %s\n", key.Path))
	details.WriteString(fmt.Sprintf("  [dim]Type:[-] %s\n", key.Type))
	if key.Size > 0 {
		details.WriteString(fmt.Sprintf("  [dim]Size:[-] %d bits\n", key.Size))
	}
	if key.Comment != "" {
		details.WriteString(fmt.Sprintf("  [dim]Comment:[-] %s\n", key.Comment))
	}
	if key.Fingerprint != "" {
		details.WriteString(fmt.Sprintf("  [dim]Fingerprint:[-] %s\n", key.Fingerprint))
	}
	details.WriteString("\n")

	// Status section
	details.WriteString("[white::b]Status:[-]\n")
	if key.LoadedInAgent {
		details.WriteString("  [green]✓[-] Loaded in ssh-agent\n")
	} else {
		details.WriteString("  [dim]○[-] Not loaded in ssh-agent\n")
	}
	if key.HasPublicKey {
		details.WriteString("  [green]✓[-] Public key (.pub) exists\n")
	} else {
		details.WriteString("  [red]✗[-] Public key (.pub) missing\n")
	}
	if key.IsEncrypted {
		details.WriteString("  [yellow]🔒[-] Encrypted (passphrase protected)\n")
	} else {
		details.WriteString("  [dim]🔓[-] Not encrypted\n")
	}
	details.WriteString("\n")

	// Commands section
	details.WriteString("[white::b]Commands:[-]\n")
	if key.LoadedInAgent {
		details.WriteString("  [yellow]u[-]: Unload from ssh-agent\n")
	} else {
		details.WriteString("  [yellow]l[-]: Load into ssh-agent\n")
	}
	// Show comment edit command for keys with a known file path
	// This includes both filesystem keys and agent keys that have been matched to their files
	if key.Path != "" {
		details.WriteString("  [yellow]C[-]: Edit comment\n")
	}

	kd.TextView.SetText(details.String())
}

func (kd *SSHKeyDetails) Clear() {
	kd.TextView.SetText("")
	kd.currentKey = nil
}

// GetCurrentKey returns the currently displayed key.
func (kd *SSHKeyDetails) GetCurrentKey() *domain.SSHKey {
	return kd.currentKey
}
