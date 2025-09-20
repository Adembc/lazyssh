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
		SetTitle(" Details ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
}

// renderTagChips builds colored tag chips for details view.
func renderTagChips(tags []string) string {
	if len(tags) == 0 {
		return ""
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
	serverKey := strings.Join(server.IdentityFiles, ", ")

	pinnedStr := "true"
	if server.PinnedAt.IsZero() {
		pinnedStr = "false"
	}
	hiddenStr := "false"
	if server.Hidden {
		hiddenStr = "true"
	}
	tagsText := renderTagChips(server.Tags)

	// Basic information
	aliasText := strings.Join(server.Aliases, ", ")

	userText := server.User

	hostText := server.Host

	portText := fmt.Sprintf("%d", server.Port)
	if server.Port == 0 {
		portText = ""
	}

	text := fmt.Sprintf(
		"[::b]%s[-]\n\n[::b]Basic Settings:[-]\n  Host: [white]%s[-]\n  User: [white]%s[-]\n  Port: [white]%s[-]\n  Key:  [white]%s[-]\n  Tags: %s\n  Pinned: [white]%s[-]\n  Hidden: [white]%s[-]\n  Last SSH: %s\n  SSH Count: [white]%d[-]\n",
		aliasText, hostText, userText, portText,
		serverKey, tagsText, pinnedStr, hiddenStr,
		lastSeen, server.SSHCount)

	// Advanced settings section (only show non-empty fields)
	// Organized by logical grouping for better readability
	type fieldEntry struct {
		name  string
		value string
	}

	type fieldGroup struct {
		name   string
		fields []fieldEntry
	}

	// Create field groups for better organization and future extensibility
	groups := []fieldGroup{
		{
			name: "Connection & Proxy",
			fields: []fieldEntry{
				{"ProxyJump", server.ProxyJump},
				{"ProxyCommand", server.ProxyCommand},
				{"RemoteCommand", server.RemoteCommand},
				{"RequestTTY", server.RequestTTY},
				{"SessionType", server.SessionType},
				{"ConnectTimeout", server.ConnectTimeout},
				{"ConnectionAttempts", server.ConnectionAttempts},
				{"BindAddress", server.BindAddress},
				{"BindInterface", server.BindInterface},
				{"AddressFamily", server.AddressFamily},
				{"ExitOnForwardFailure", server.ExitOnForwardFailure},
				{"IPQoS", server.IPQoS},
				{"CanonicalizeHostname", server.CanonicalizeHostname},
				{"CanonicalDomains", server.CanonicalDomains},
				{"CanonicalizeFallbackLocal", server.CanonicalizeFallbackLocal},
				{"CanonicalizeMaxDots", server.CanonicalizeMaxDots},
				{"CanonicalizePermittedCNAMEs", server.CanonicalizePermittedCNAMEs},
				{"ServerAliveInterval", server.ServerAliveInterval},
				{"ServerAliveCountMax", server.ServerAliveCountMax},
				{"Compression", server.Compression},
				{"TCPKeepAlive", server.TCPKeepAlive},
				{"BatchMode", server.BatchMode},
				{"ControlMaster", server.ControlMaster},
				{"ControlPath", server.ControlPath},
				{"ControlPersist", server.ControlPersist},
			},
		},
		{
			name: "Authentication",
			fields: []fieldEntry{
				{"PubkeyAuthentication", server.PubkeyAuthentication},
				{"PubkeyAcceptedAlgorithms", server.PubkeyAcceptedAlgorithms},
				{"HostbasedAcceptedAlgorithms", server.HostbasedAcceptedAlgorithms},
				{"PasswordAuthentication", server.PasswordAuthentication},
				{"PreferredAuthentications", server.PreferredAuthentications},
				{"IdentitiesOnly", server.IdentitiesOnly},
				{"AddKeysToAgent", server.AddKeysToAgent},
				{"IdentityAgent", server.IdentityAgent},
				{"KbdInteractiveAuthentication", server.KbdInteractiveAuthentication},
				{"NumberOfPasswordPrompts", server.NumberOfPasswordPrompts},
			},
		},
		{
			name: "Forwarding",
			fields: []fieldEntry{
				{"ForwardAgent", server.ForwardAgent},
				{"ForwardX11", server.ForwardX11},
				{"ForwardX11Trusted", server.ForwardX11Trusted},
				{"LocalForward", strings.Join(server.LocalForward, ", ")},
				{"RemoteForward", strings.Join(server.RemoteForward, ", ")},
				{"DynamicForward", strings.Join(server.DynamicForward, ", ")},
				{"ClearAllForwardings", server.ClearAllForwardings},
				{"GatewayPorts", server.GatewayPorts},
			},
		},
		{
			name: "Security & Cryptography",
			fields: []fieldEntry{
				{"StrictHostKeyChecking", server.StrictHostKeyChecking},
				{"CheckHostIP", server.CheckHostIP},
				{"FingerprintHash", server.FingerprintHash},
				{"UserKnownHostsFile", server.UserKnownHostsFile},
				{"HostKeyAlgorithms", server.HostKeyAlgorithms},
				{"Ciphers", server.Ciphers},
				{"MACs", server.MACs},
				{"KexAlgorithms", server.KexAlgorithms},
				{"VerifyHostKeyDNS", server.VerifyHostKeyDNS},
				{"UpdateHostKeys", server.UpdateHostKeys},
				{"HashKnownHosts", server.HashKnownHosts},
				{"VisualHostKey", server.VisualHostKey},
			},
		},
		{
			name: "Environment & Execution",
			fields: []fieldEntry{
				{"LocalCommand", server.LocalCommand},
				{"PermitLocalCommand", server.PermitLocalCommand},
				{"EscapeChar", server.EscapeChar},
				{"SendEnv", strings.Join(server.SendEnv, ", ")},
				{"SetEnv", strings.Join(server.SetEnv, ", ")},
			},
		},
		{
			name: "Debugging",
			fields: []fieldEntry{
				{"LogLevel", server.LogLevel},
			},
		},
	}

	// Build advanced settings text without group labels for cleaner display
	hasAdvanced := false
	advancedText := "\n[::b]Advanced Settings:[-]\n"

	for _, group := range groups {
		for _, field := range group.fields {
			if field.value != "" {
				hasAdvanced = true
				advancedText += fmt.Sprintf("  %s: [white]%s[-]\n", field.name, field.value)
			}
		}
	}

	if hasAdvanced {
		text += advancedText
	}

	// Commands list
	text += "\n[::b]Commands:[-]\n  Enter: SSH connect\n  c: Copy SSH command\n  g: Ping server\n  r: Refresh list\n  a: Add new server\n  e: Edit entry\n  t: Edit tags\n  d: Delete entry\n  p: Pin/Unpin\n  h: Hide/Unhide\n  H: Toggle hidden view"

	sd.TextView.SetText(text)
}

func (sd *ServerDetails) ShowEmpty() {
	sd.TextView.SetText("No servers match the current filter.")
}
