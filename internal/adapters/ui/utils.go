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

// SSH config value constants
const (
	sshYes   = "yes"
	sshNo    = "no"
	sshForce = "force"
	sshAuto  = "auto"
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
// Format: ssh [options] [user@]host [command]
func BuildSSHCommand(s domain.Server) string {
	parts := []string{"ssh"}

	// Add proxy and connection options
	addProxyOptions(&parts, s)
	addConnectionTimingOptions(&parts, s)

	// Add port forwarding options
	addPortForwardingOptions(&parts, s)

	// Add authentication options
	addAuthOptions(&parts, s)

	// Add agent and forwarding options
	addForwardingOptions(&parts, s)

	// Add connection multiplexing options
	addMultiplexingOptions(&parts, s)

	// Add connection reliability options
	addConnectionOptions(&parts, s)

	// Add security options
	addSecurityOptions(&parts, s)

	// Add command execution options
	addCommandExecutionOptions(&parts, s)

	// Add environment options
	addEnvironmentOptions(&parts, s)

	// Add TTY and logging options
	addTTYAndLoggingOptions(&parts, s)

	// Port option
	if s.Port != 0 && s.Port != 22 {
		parts = append(parts, "-p", fmt.Sprintf("%d", s.Port))
	}

	// Identity file option
	if len(s.IdentityFiles) > 0 {
		for _, keyFile := range s.IdentityFiles {
			parts = append(parts, "-i", quoteIfNeeded(keyFile))
		}
	}

	// Host specification
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

	// RemoteCommand (must come after the host)
	if s.RemoteCommand != "" {
		parts = append(parts, quoteIfNeeded(s.RemoteCommand))
	}

	return strings.Join(parts, " ")
}

// addProxyOptions adds proxy-related options to the SSH command
func addProxyOptions(parts *[]string, s domain.Server) {
	if s.ProxyJump != "" {
		*parts = append(*parts, "-J", quoteIfNeeded(s.ProxyJump))
	}
	if s.ProxyCommand != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ProxyCommand=%s", quoteIfNeeded(s.ProxyCommand)))
	}
}

// addConnectionTimingOptions adds connection timing options to the SSH command
func addConnectionTimingOptions(parts *[]string, s domain.Server) {
	if s.ConnectTimeout != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ConnectTimeout=%s", s.ConnectTimeout))
	}
	if s.ConnectionAttempts != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ConnectionAttempts=%s", s.ConnectionAttempts))
	}
	if s.BindAddress != "" {
		*parts = append(*parts, "-b", s.BindAddress)
	}
	if s.BindInterface != "" {
		*parts = append(*parts, "-B", s.BindInterface)
	}
}

// addPortForwardingOptions adds port forwarding options to the SSH command
func addPortForwardingOptions(parts *[]string, s domain.Server) {
	for _, forward := range s.LocalForward {
		*parts = append(*parts, "-L", forward)
	}
	for _, forward := range s.RemoteForward {
		*parts = append(*parts, "-R", forward)
	}
	for _, forward := range s.DynamicForward {
		*parts = append(*parts, "-D", forward)
	}
}

// addAuthOptions adds authentication-related options to the SSH command
func addAuthOptions(parts *[]string, s domain.Server) {
	if s.PubkeyAuthentication != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("PubkeyAuthentication=%s", s.PubkeyAuthentication))
	}
	if s.PubkeyAcceptedAlgorithms != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("PubkeyAcceptedAlgorithms=%s", s.PubkeyAcceptedAlgorithms))
	}
	if s.HostbasedAcceptedAlgorithms != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("HostbasedAcceptedAlgorithms=%s", s.HostbasedAcceptedAlgorithms))
	}
	if s.PasswordAuthentication != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("PasswordAuthentication=%s", s.PasswordAuthentication))
	}
	if s.PreferredAuthentications != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("PreferredAuthentications=%s", s.PreferredAuthentications))
	}
	if s.IdentitiesOnly != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("IdentitiesOnly=%s", s.IdentitiesOnly))
	}
	if s.AddKeysToAgent != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("AddKeysToAgent=%s", s.AddKeysToAgent))
	}
	if s.IdentityAgent != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("IdentityAgent=%s", quoteIfNeeded(s.IdentityAgent)))
	}
}

// addForwardingOptions adds agent and X11 forwarding options to the SSH command
func addForwardingOptions(parts *[]string, s domain.Server) {
	if s.ForwardAgent != "" {
		if s.ForwardAgent == sshYes {
			*parts = append(*parts, "-A")
		} else if s.ForwardAgent == sshNo {
			*parts = append(*parts, "-a")
		}
	}
	if s.ForwardX11 != "" {
		if s.ForwardX11 == sshYes {
			*parts = append(*parts, "-X")
		} else if s.ForwardX11 == sshNo {
			*parts = append(*parts, "-x")
		}
	}
	if s.ForwardX11Trusted == sshYes {
		*parts = append(*parts, "-Y")
	}
}

// addMultiplexingOptions adds connection multiplexing options to the SSH command
func addMultiplexingOptions(parts *[]string, s domain.Server) {
	if s.ControlMaster != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ControlMaster=%s", s.ControlMaster))
	}
	if s.ControlPath != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ControlPath=%s", quoteIfNeeded(s.ControlPath)))
	}
	if s.ControlPersist != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ControlPersist=%s", s.ControlPersist))
	}
}

// addConnectionOptions adds connection reliability options to the SSH command
func addConnectionOptions(parts *[]string, s domain.Server) {
	if s.ServerAliveInterval != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ServerAliveInterval=%s", s.ServerAliveInterval))
	}
	if s.ServerAliveCountMax != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("ServerAliveCountMax=%s", s.ServerAliveCountMax))
	}
	if s.Compression == sshYes {
		*parts = append(*parts, "-C")
	}
	if s.TCPKeepAlive != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("TCPKeepAlive=%s", s.TCPKeepAlive))
	}
}

// addCommandExecutionOptions adds command execution options to the SSH command
func addCommandExecutionOptions(parts *[]string, s domain.Server) {
	if s.LocalCommand != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("LocalCommand=%s", quoteIfNeeded(s.LocalCommand)))
	}
	if s.PermitLocalCommand != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("PermitLocalCommand=%s", s.PermitLocalCommand))
	}
}

// addEnvironmentOptions adds environment variable options to the SSH command
func addEnvironmentOptions(parts *[]string, s domain.Server) {
	for _, env := range s.SendEnv {
		*parts = append(*parts, "-o", fmt.Sprintf("SendEnv=%s", env))
	}
	for _, env := range s.SetEnv {
		*parts = append(*parts, "-o", fmt.Sprintf("SetEnv=%s", quoteIfNeeded(env)))
	}
}

// addSecurityOptions adds security-related options to the SSH command
func addSecurityOptions(parts *[]string, s domain.Server) {
	if s.StrictHostKeyChecking != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("StrictHostKeyChecking=%s", s.StrictHostKeyChecking))
	}
	if s.UserKnownHostsFile != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("UserKnownHostsFile=%s", quoteIfNeeded(s.UserKnownHostsFile)))
	}
	if s.HostKeyAlgorithms != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("HostKeyAlgorithms=%s", s.HostKeyAlgorithms))
	}
	if s.MACs != "" {
		*parts = append(*parts, "-m", s.MACs)
	}
	if s.Ciphers != "" {
		*parts = append(*parts, "-c", s.Ciphers)
	}
	if s.KexAlgorithms != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("KexAlgorithms=%s", s.KexAlgorithms))
	}
}

// addTTYAndLoggingOptions adds TTY and logging options to the SSH command
func addTTYAndLoggingOptions(parts *[]string, s domain.Server) {
	// RequestTTY option
	if s.RequestTTY != "" {
		switch s.RequestTTY {
		case sshYes:
			*parts = append(*parts, "-t")
		case sshNo:
			*parts = append(*parts, "-T")
		case sshForce:
			*parts = append(*parts, "-tt")
		case sshAuto:
			// auto is the default behavior, no flag needed
		default:
			// For any other value, pass it as-is via -o
			*parts = append(*parts, "-o", fmt.Sprintf("RequestTTY=%s", s.RequestTTY))
		}
	}

	// LogLevel option
	if s.LogLevel != "" {
		switch strings.ToLower(s.LogLevel) {
		case "quiet":
			*parts = append(*parts, "-q")
		case "verbose":
			*parts = append(*parts, "-v")
		case "debug", "debug1":
			*parts = append(*parts, "-v")
		case "debug2":
			*parts = append(*parts, "-vv")
		case "debug3":
			*parts = append(*parts, "-vvv")
		}
	}

	// BatchMode option
	if s.BatchMode == sshYes {
		*parts = append(*parts, "-o", "BatchMode=yes")
	}
}

// quoteIfNeeded returns the value quoted if it contains spaces.
func quoteIfNeeded(val string) string {
	if strings.ContainsAny(val, " \t") {
		return fmt.Sprintf("%q", val)
	}
	return val
}
