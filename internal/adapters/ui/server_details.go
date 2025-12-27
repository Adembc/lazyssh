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
	"os"
	"path/filepath"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerDetails struct {
	*tview.TextView
	gitService ports.GitService
	serverRepo ports.ServerRepository
}

func NewServerDetails(gitService ports.GitService, serverRepo ports.ServerRepository) *ServerDetails {
	details := &ServerDetails{
		TextView:   tview.NewTextView(),
		gitService: gitService,
		serverRepo: serverRepo,
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

// getSSHKeyForServer attempts to fetch SSH key details for the server's identity file
func (sd *ServerDetails) getSSHKeyForServer(server domain.Server) *domain.SSHKey {
	if sd.gitService == nil || sd.serverRepo == nil {
		return nil
	}

	// Get the first identity file if available
	if len(server.IdentityFiles) == 0 {
		return nil
	}

	identityFile := server.IdentityFiles[0]

	// Expand ~ to home directory for comparison
	if strings.HasPrefix(identityFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			identityFile = filepath.Join(homeDir, identityFile[2:])
		}
	} else if identityFile == "~" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			identityFile = homeDir
		}
	}

	// Fetch all SSH keys and find the one matching this identity file
	allKeys, err := sd.gitService.ListAllSSHKeys(sd.serverRepo)
	if err != nil {
		return nil
	}

	for _, key := range allKeys {
		if key.Path == identityFile {
			return &key
		}
	}

	return nil
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
		"[::b]%s[-]\n\n[::b]Basic Settings:[-]\n  Host: [white]%s[-]\n  User: [white]%s[-]\n  Port: [white]%s[-]\n  Key:  [white]%s[-]\n  Tags: %s\n  Pinned: [white]%s[-]\n  Last SSH: %s\n  SSH Count: [white]%d[-]\n",
		aliasText, hostText, userText, portText,
		serverKey, tagsText, pinnedStr,
		lastSeen, server.SSHCount)

	// Add SSH Key Details section if key is configured
	if sshKey := sd.getSSHKeyForServer(server); sshKey != nil {
		text += "\n[::b]SSH Key Details:[-]\n"
		text += fmt.Sprintf("  [dim]Path:[-] %s\n", sshKey.Path)
		text += fmt.Sprintf("  [dim]Type:[-] %s\n", sshKey.Type)
		if sshKey.Size > 0 {
			text += fmt.Sprintf("  [dim]Size:[-] %d bits\n", sshKey.Size)
		}
		if sshKey.Comment != "" {
			text += fmt.Sprintf("  [dim]Comment:[-] %s\n", sshKey.Comment)
		}
		text += "\n[::b]  Status:[-]\n"
		if sshKey.LoadedInAgent {
			text += "    [green]✓[-] Loaded in ssh-agent\n"
		} else {
			text += "    [dim]○[-] Not loaded in ssh-agent\n"
		}
		if sshKey.HasPublicKey {
			text += "    [green]✓[-] Public key (.pub) exists\n"
		} else {
			text += "    [red]✗[-] Public key (.pub) missing\n"
		}
		if sshKey.IsEncrypted {
			text += "    [yellow]🔒[-] Encrypted (passphrase protected)\n"
		} else {
			text += "    [dim]🔓[-] Not encrypted\n"
		}
		text += "\n[::b]  Commands:[-]\n"
		if sshKey.LoadedInAgent {
			text += "    [yellow]u[-]: Unload from ssh-agent\n"
		} else {
			text += "    [yellow]l[-]: Load into ssh-agent\n"
		}
		text += "    [yellow]C[-]: Edit comment\n"
	}

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
	text += "\n[::b]Commands:[-]\n  [yellow]Enter[-]: SSH connect\n  [yellow]f[-]: Port forward\n  [yellow]x[-]: Stop forwarding\n  [yellow]c[-]: Copy SSH command\n  [yellow]g[-]: Ping server\n  [yellow]r[-]: Refresh list\n  [yellow]a[-]: Add new server\n  [yellow]e[-]: Edit entry\n  [yellow]t[-]: Edit tags\n  [yellow]d[-]: Delete entry\n  [yellow]p[-]: Pin/Unpin"

	sd.TextView.SetText(text)
}

func (sd *ServerDetails) ShowEmpty() {
	sd.TextView.SetText("No servers match the current filter.")
}
