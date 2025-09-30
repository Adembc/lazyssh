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

	// SessionType values
	sessionTypeNone      = "none"
	sessionTypeSubsystem = "subsystem"
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
	// Start with alias and tags comment for easy parsing when pasting
	comment := "# lazyssh-alias:" + s.Alias
	if len(s.Tags) > 0 {
		comment += " tags:" + strings.Join(s.Tags, ",")
	}
	parts := []string{comment, "\nssh"}

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
		// Handle special case: RemoteCommand=none clears the command (OpenSSH 7.6+)
		if s.RemoteCommand == sessionTypeNone {
			parts = append(parts, "-o", "RemoteCommand=none")
		} else {
			parts = append(parts, quoteIfNeeded(s.RemoteCommand))
		}
	}

	return strings.Join(parts, " ")
}

// addOption adds an SSH option in the format "-o Key=Value" if value is not empty
func addOption(parts *[]string, key, value string) {
	if value != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("%s=%s", key, value))
	}
}

// addQuotedOption adds an SSH option with quoted value if needed
func addQuotedOption(parts *[]string, key, value string) {
	if value != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("%s=%s", key, quoteIfNeeded(value)))
	}
}

// addProxyOptions adds proxy-related options to the SSH command
func addProxyOptions(parts *[]string, s domain.Server) {
	if s.ProxyJump != "" {
		*parts = append(*parts, "-J", quoteIfNeeded(s.ProxyJump))
	}
	addQuotedOption(parts, "ProxyCommand", s.ProxyCommand)
}

// addConnectionTimingOptions adds connection timing options to the SSH command
func addConnectionTimingOptions(parts *[]string, s domain.Server) {
	addOption(parts, "ConnectTimeout", s.ConnectTimeout)
	addOption(parts, "ConnectionAttempts", s.ConnectionAttempts)
	if s.BindAddress != "" {
		*parts = append(*parts, "-b", s.BindAddress)
	}
	if s.BindInterface != "" {
		*parts = append(*parts, "-B", s.BindInterface)
	}
	addOption(parts, "AddressFamily", s.AddressFamily)
	addOption(parts, "IPQoS", s.IPQoS)
	// Hostname canonicalization options
	addOption(parts, "CanonicalizeHostname", s.CanonicalizeHostname)
	addOption(parts, "CanonicalDomains", s.CanonicalDomains)
	addOption(parts, "CanonicalizeFallbackLocal", s.CanonicalizeFallbackLocal)
	addOption(parts, "CanonicalizeMaxDots", s.CanonicalizeMaxDots)
	addQuotedOption(parts, "CanonicalizePermittedCNAMEs", s.CanonicalizePermittedCNAMEs)
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
	if s.ClearAllForwardings == sshYes {
		*parts = append(*parts, "-o", "ClearAllForwardings=yes")
	}
	if s.ExitOnForwardFailure == sshYes {
		*parts = append(*parts, "-o", "ExitOnForwardFailure=yes")
	}
	if s.GatewayPorts != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("GatewayPorts=%s", s.GatewayPorts))
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
	if s.KbdInteractiveAuthentication != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("KbdInteractiveAuthentication=%s", s.KbdInteractiveAuthentication))
	}
	if s.NumberOfPasswordPrompts != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("NumberOfPasswordPrompts=%s", s.NumberOfPasswordPrompts))
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
	if s.BatchMode == sshYes {
		*parts = append(*parts, "-o", "BatchMode=yes")
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
	if s.EscapeChar != "" {
		*parts = append(*parts, "-e", s.EscapeChar)
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
	if s.CheckHostIP != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("CheckHostIP=%s", s.CheckHostIP))
	}
	if s.FingerprintHash != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("FingerprintHash=%s", s.FingerprintHash))
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
	if s.VerifyHostKeyDNS != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("VerifyHostKeyDNS=%s", s.VerifyHostKeyDNS))
	}
	if s.UpdateHostKeys != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("UpdateHostKeys=%s", s.UpdateHostKeys))
	}
	if s.HashKnownHosts != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("HashKnownHosts=%s", s.HashKnownHosts))
	}
	if s.VisualHostKey != "" {
		*parts = append(*parts, "-o", fmt.Sprintf("VisualHostKey=%s", s.VisualHostKey))
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

	// SessionType option (OpenSSH 8.7+)
	// "none" is equivalent to -N flag
	if s.SessionType != "" {
		switch s.SessionType {
		case sessionTypeNone:
			// Use -N flag for better compatibility
			*parts = append(*parts, "-N")
		case sessionTypeSubsystem:
			// Use -s flag for subsystem
			*parts = append(*parts, "-s")
		default:
			// For other values, use -o SessionType=
			*parts = append(*parts, "-o", fmt.Sprintf("SessionType=%s", s.SessionType))
		}
	}
}

// quoteIfNeeded returns the value quoted if it contains spaces.
func quoteIfNeeded(val string) string {
	if strings.ContainsAny(val, " \t") {
		return fmt.Sprintf("%q", val)
	}
	return val
}

// GetAvailableSSHKeys returns a list of available SSH private key files in the user's .ssh directory.
// It safely handles file permission issues and only returns readable key files.
func GetAvailableSSHKeys() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	keys := []string{}

	// Common SSH key filenames to look for
	commonKeyFiles := []string{
		"id_ed25519",
		"id_rsa",
		"id_ecdsa",
		"id_dsa",
		"id_ed25519_sk",
		"id_ecdsa_sk",
	}

	// First, add common key files if they exist and are readable
	for _, keyName := range commonKeyFiles {
		keyPath := filepath.Join(sshDir, keyName)
		if info, err := os.Stat(keyPath); err == nil && !info.IsDir() {
			// Check if file is readable
			// #nosec G304 - keyPath is constructed from known safe values
			if file, err := os.Open(keyPath); err == nil {
				_ = file.Close()
				// Use tilde notation for user-friendly display
				keys = append(keys, "~/.ssh/"+keyName)
			}
		}
	}

	// Additionally, scan for any other files that look like private keys
	// (files without .pub extension and with appropriate permissions)
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		return keys // Return what we have so far
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Skip public keys, known_hosts, config, and authorized_keys
		if strings.HasSuffix(name, ".pub") ||
			name == "known_hosts" ||
			name == "config" ||
			name == "authorized_keys" ||
			strings.HasPrefix(name, ".") {
			continue
		}

		// Skip if already added as common key
		isCommon := false
		for _, commonKey := range commonKeyFiles {
			if name == commonKey {
				isCommon = true
				break
			}
		}
		if isCommon {
			continue
		}

		// Check if file is readable and looks like a private key
		keyPath := filepath.Join(sshDir, name)
		// #nosec G304 - keyPath is constructed from safe directory enumeration
		if file, err := os.Open(keyPath); err == nil {
			// Read first few bytes to check if it might be a private key
			buffer := make([]byte, 100)
			n, readErr := file.Read(buffer)
			_ = file.Close()
			if readErr == nil && n > 0 {
				content := string(buffer[:n])
				if strings.Contains(content, "PRIVATE KEY") ||
					strings.Contains(content, "SSH2 ENCRYPTED PRIVATE KEY") {
					keys = append(keys, "~/.ssh/"+name)
				}
			}
		}
	}

	return keys
}

// GetAvailableKnownHostsFiles returns a list of available known_hosts files in common locations.
// It safely handles file permission issues and only returns readable files.
func GetAvailableKnownHostsFiles() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}
	}

	files := []string{}

	// Common known_hosts file locations to check
	commonPaths := []string{
		filepath.Join(homeDir, ".ssh", "known_hosts"),
		filepath.Join(homeDir, ".ssh", "known_hosts2"),
		filepath.Join(homeDir, ".ssh", "known_hosts.old"),
		"/etc/ssh/ssh_known_hosts",
		"/etc/ssh/ssh_known_hosts2",
	}

	// Check each common location
	for _, path := range commonPaths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// Check if file is readable
			// #nosec G304 - path is constructed from known safe values
			if file, err := os.Open(path); err == nil {
				_ = file.Close()
				// Convert to user-friendly path with tilde notation
				if strings.HasPrefix(path, homeDir) {
					relPath := strings.TrimPrefix(path, homeDir)
					files = append(files, "~"+relPath)
				} else {
					files = append(files, path)
				}
			}
		}
	}

	// Also check for any other files in .ssh directory that might be known_hosts files
	sshDir := filepath.Join(homeDir, ".ssh")
	entries, err := os.ReadDir(sshDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Look for files that might be known_hosts variants
			if strings.Contains(name, "known_hosts") && !strings.HasSuffix(name, ".pub") {
				// Skip if already added from common paths
				fullPath := filepath.Join(sshDir, name)
				tildeNotation := "~/.ssh/" + name

				alreadyAdded := false
				for _, existing := range files {
					if existing == tildeNotation {
						alreadyAdded = true
						break
					}
				}

				if !alreadyAdded {
					// Check if readable
					// #nosec G304 - fullPath is constructed from safe directory enumeration
					if file, err := os.Open(fullPath); err == nil {
						_ = file.Close()
						files = append(files, tildeNotation)
					}
				}
			}
		}
	}

	return files
}
