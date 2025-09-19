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
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

// Log level constants
const (
	logVerbose = "VERBOSE"
	logDebug   = "DEBUG"
	logDebug2  = "DEBUG2"
	logDebug3  = "DEBUG3"
	logQuiet   = "QUIET"
	sshUser    = "user"
)

// ParseSSHCommand parses an SSH command string into a domain.Server struct
// Supports standard SSH command arguments and -o options mapping to ssh_config
// Also supports multiline commands with backslash line continuation
func ParseSSHCommand(cmd string) (*domain.Server, error) {
	// Remove leading/trailing whitespace first
	cmd = strings.TrimSpace(cmd)

	// Extract alias and tags from comment if present (do this before preprocessing multiline)
	var extractedAlias, extractedTags string
	lines := strings.Split(cmd, "\n")
	cmdLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Check for any comment line
		if strings.HasPrefix(trimmed, "#") {
			// Check if it's a lazyssh metadata comment
			if strings.Contains(trimmed, "lazyssh-alias:") {
				// Parse alias
				parts := strings.Split(trimmed, "lazyssh-alias:")
				if len(parts) >= 2 {
					aliasAndTags := strings.TrimSpace(parts[1])
					// Check if tags are included
					if tagIdx := strings.Index(aliasAndTags, " tags:"); tagIdx > 0 {
						extractedAlias = strings.TrimSpace(aliasAndTags[:tagIdx])
						extractedTags = strings.TrimSpace(aliasAndTags[tagIdx+6:]) // 6 is length of " tags:"
					} else {
						extractedAlias = aliasAndTags
					}
				}
			}
			// Skip all comment lines (both metadata and regular comments)
			continue
		}
		// Keep non-comment lines
		cmdLines = append(cmdLines, line)
	}
	cmd = strings.Join(cmdLines, "\n")

	// Now preprocess multiline commands (handle backslash line continuation)
	cmd = preprocessMultilineCommand(cmd)
	cmd = strings.TrimSpace(cmd)

	// Check if it starts with "ssh"
	if !strings.HasPrefix(cmd, "ssh ") && cmd != "ssh" {
		return nil, fmt.Errorf("not a valid ssh command")
	}

	// Split the command into parts, preserving quoted strings
	parts, err := splitCommand(cmd)
	if err != nil {
		return nil, err
	}

	if len(parts) < 2 {
		return nil, fmt.Errorf("missing host specification")
	}

	// Remove "ssh" from the beginning
	parts = parts[1:]

	server := &domain.Server{}

	// Parse all parts: flags can appear before or after the destination
	if err := parseSSHCommandParts(server, parts); err != nil {
		return nil, err
	}

	// Override alias and tags if extracted from comment
	if extractedAlias != "" {
		server.Alias = extractedAlias
	}
	if extractedTags != "" {
		// Split tags by comma
		server.Tags = strings.Split(extractedTags, ",")
		// Trim spaces from each tag
		for i, tag := range server.Tags {
			server.Tags[i] = strings.TrimSpace(tag)
		}
	}

	// Set default port if not specified
	if server.Port == 0 {
		server.Port = 22
	}

	return server, nil
}

// parseSSHCommandParts parses all SSH command parts including flags and destination
func parseSSHCommandParts(server *domain.Server, parts []string) error {
	destIndex := -1
	flagIndices := make(map[int]bool)
	var remoteCommandParts []string

	// First pass: find the destination (first non-flag argument)
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if !strings.HasPrefix(part, "-") {
			destIndex = i
			break
		}
		// Skip flag values
		consumed, _ := peekFlagConsumption(parts, i)
		if consumed > 1 {
			i += consumed - 1
		}
	}

	if destIndex == -1 {
		return fmt.Errorf("missing destination host")
	}

	// Parse the destination
	dest := parts[destIndex]
	if strings.Contains(dest, "@") {
		userHost := strings.SplitN(dest, "@", 2)
		server.User = userHost[0]
		dest = userHost[1]
	}

	// Parse host:port format
	if strings.Contains(dest, ":") && !strings.Contains(dest, "[") {
		hostPort := strings.SplitN(dest, ":", 2)
		server.Host = hostPort[0]
		port, err := strconv.Atoi(hostPort[1])
		if err == nil {
			server.Port = port
		}
	} else {
		server.Host = dest
	}

	// Second pass: parse all flags (before and after destination) and collect remote command
	i := 0
	afterDestination := false
	for i < len(parts) {
		if i == destIndex {
			// Skip the destination
			flagIndices[i] = true
			afterDestination = true
			i++
			continue
		}

		part := parts[i]

		// Check if this is a flag
		if strings.HasPrefix(part, "-") {
			// After destination, only parse known SSH flags
			if afterDestination {
				// Check if it's a known SSH flag
				consumed, _ := peekFlagConsumption(parts, i)
				if consumed > 0 {
					// It's a known SSH flag, parse it
					actualConsumed, err := parseFlag(server, parts, i)
					if err != nil {
						return err
					}
					// Mark all consumed indices as flag indices
					for j := 0; j < actualConsumed; j++ {
						flagIndices[i+j] = true
					}
					i += actualConsumed
				} else {
					// Unknown flag after destination - probably part of remote command
					i++
				}
			} else {
				// Before destination, parse all flags
				consumed, err := parseFlag(server, parts, i)
				if err != nil {
					return err
				}
				if consumed == 0 {
					consumed = 1
				}
				// Mark all consumed indices as flag indices
				for j := 0; j < consumed; j++ {
					flagIndices[i+j] = true
				}
				i += consumed
			}
		} else {
			// This might be part of remote command
			i++
		}
	}

	// Collect remote command (anything that's not a flag or destination)
	for i, part := range parts {
		if !flagIndices[i] && i != destIndex {
			remoteCommandParts = append(remoteCommandParts, part)
		}
	}

	if len(remoteCommandParts) > 0 {
		server.RemoteCommand = strings.Join(remoteCommandParts, " ")
	}

	// Set default alias if not set
	if server.Alias == "" {
		server.Alias = GenerateSmartAlias(server.Host, server.User, server.Port)
	}

	return nil
}

// peekFlagConsumption checks how many parts a flag would consume without modifying state
func peekFlagConsumption(parts []string, i int) (int, error) {
	if i >= len(parts) {
		return 0, nil
	}

	part := parts[i]

	// Flags that consume 2 parts (flag + value)
	twoPartFlags := map[string]bool{
		"-p": true, "-l": true, "-i": true, "-b": true, "-B": true,
		"-J": true, "-W": true, "-L": true, "-R": true, "-D": true,
		"-c": true, "-m": true, "-e": true, "-F": true, "-S": true,
		"-O": true, "-o": true,
	}

	if twoPartFlags[part] {
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing value after %s", part)
		}
		return 2, nil
	}

	// Single-part SSH flags
	singlePartFlags := map[string]bool{
		"-4": true, "-6": true, "-A": true, "-a": true, "-C": true,
		"-f": true, "-g": true, "-k": true, "-K": true, "-M": true,
		"-n": true, "-N": true, "-q": true, "-s": true, "-T": true,
		"-t": true, "-v": true, "-V": true, "-x": true, "-X": true,
		"-Y": true,
		// Also include verbose variations
		"-vv": true, "-vvv": true,
	}

	if singlePartFlags[part] {
		return 1, nil
	}

	// Unknown flag - return 0
	return 0, nil
}

// parseFlag parses a single flag and returns how many parts were consumed
func parseFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	// Try parsing different categories of flags
	if consumed, err := parseConnectionFlag(server, parts, i); err != nil {
		return 0, err
	} else if consumed > 0 {
		return consumed, nil
	}

	if consumed, err := parseAuthenticationFlag(server, parts, i); err != nil {
		return 0, err
	} else if consumed > 0 {
		return consumed, nil
	}

	if consumed, err := parseForwardingFlag(server, parts, i); err != nil {
		return 0, err
	} else if consumed > 0 {
		return consumed, nil
	}

	if consumed, err := parseProxyFlag(server, parts, i); err != nil {
		return 0, err
	} else if consumed > 0 {
		return consumed, nil
	}

	if consumed := parseLoggingFlag(server, parts, i); consumed > 0 {
		return consumed, nil
	}

	if consumed, err := parseMiscFlag(server, parts, i); err != nil {
		return 0, err
	} else if consumed > 0 {
		return consumed, nil
	}

	// Handle -o option specially
	if part == "-o" {
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing option value after -o")
		}
		if err := parseSSHOption(server, parts[i+1]); err != nil {
			return 0, fmt.Errorf("invalid SSH option: %w", err)
		}
		return 2, nil // -o consumes 2 parts: the flag and its value
	}

	// Unknown flag - return 1 to consume only this flag
	if strings.HasPrefix(part, "-") && len(part) > 1 {
		return 1, nil
	}

	return 0, nil
}

// parseConnectionFlag parses connection-related flags
func parseConnectionFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	switch part {
	case "-p":
		// Port
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing port value after -p")
		}
		port, err := strconv.Atoi(parts[i+1])
		if err != nil {
			return 0, fmt.Errorf("invalid port value: %s", parts[i+1])
		}
		server.Port = port
		return 2, nil

	case "-4":
		// Force IPv4
		server.AddressFamily = "inet"
		return 1, nil

	case "-6":
		// Force IPv6
		server.AddressFamily = "inet6"
		return 1, nil

	case "-b":
		// Bind address
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing bind address after -b")
		}
		server.BindAddress = parts[i+1]
		return 2, nil

	case "-B":
		// Bind interface (OpenSSH 8.9+)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing bind interface after -B")
		}
		server.BindInterface = parts[i+1]
		return 2, nil

	case "-C":
		// Enable compression
		server.Compression = sshYes
		return 1, nil

	case "-N":
		// No remote command
		server.SessionType = "none"
		return 1, nil

	case "-T":
		// Disable pseudo-terminal allocation
		server.RequestTTY = sshNo
		return 1, nil

	case "-t":
		// Force pseudo-terminal allocation
		server.RequestTTY = sshYes
		return 1, nil

	case "-n":
		// Redirect stdin from /dev/null (batch mode)
		server.BatchMode = sshYes
		return 1, nil

	case "-s":
		// Subsystem
		server.SessionType = "subsystem"
		return 1, nil

	case "-f":
		// Go to background (runtime behavior, not a config option)
		return 1, nil

	case "-g":
		// Allow remote hosts to connect to forwarded ports
		server.GatewayPorts = sshYes
		return 1, nil
	}

	return 0, nil
}

// parseAuthenticationFlag parses authentication-related flags
func parseAuthenticationFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	switch part {
	case "-l":
		// Login name (user)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing user value after -l")
		}
		server.User = parts[i+1]
		return 2, nil

	case "-i":
		// Identity file
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing identity file after -i")
		}
		if server.IdentityFiles == nil {
			server.IdentityFiles = []string{}
		}
		server.IdentityFiles = append(server.IdentityFiles, parts[i+1])
		return 2, nil

	case "-A":
		// Forward agent
		server.ForwardAgent = sshYes
		return 1, nil

	case "-a":
		// Disable agent forwarding
		server.ForwardAgent = sshNo
		return 1, nil

	case "-k":
		// Disable GSSAPI authentication
		// We'll skip GSSAPI-related options
		return 1, nil

	case "-K":
		// Enable GSSAPI authentication and forwarding
		// We'll skip GSSAPI-related options
		return 1, nil
	}

	return 0, nil
}

// parseForwardingFlag parses forwarding-related flags
func parseForwardingFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	switch part {
	case "-L":
		// Local port forwarding
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing local forward specification after -L")
		}
		if server.LocalForward == nil {
			server.LocalForward = []string{}
		}
		server.LocalForward = append(server.LocalForward, parts[i+1])
		return 2, nil

	case "-R":
		// Remote port forwarding
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing remote forward specification after -R")
		}
		if server.RemoteForward == nil {
			server.RemoteForward = []string{}
		}
		server.RemoteForward = append(server.RemoteForward, parts[i+1])
		return 2, nil

	case "-D":
		// Dynamic port forwarding (SOCKS)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing dynamic forward specification after -D")
		}
		if server.DynamicForward == nil {
			server.DynamicForward = []string{}
		}
		server.DynamicForward = append(server.DynamicForward, parts[i+1])
		return 2, nil

	case "-X":
		// Enable X11 forwarding
		server.ForwardX11 = sshYes
		return 1, nil

	case "-x":
		// Disable X11 forwarding
		server.ForwardX11 = sshNo
		return 1, nil

	case "-Y":
		// Enable trusted X11 forwarding
		server.ForwardX11 = sshYes
		server.ForwardX11Trusted = sshYes
		return 1, nil
	}

	return 0, nil
}

// parseProxyFlag parses proxy-related flags
func parseProxyFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	switch part {
	case "-J":
		// ProxyJump
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing proxy jump value after -J")
		}
		server.ProxyJump = parts[i+1]
		return 2, nil

	case "-W":
		// ProxyCommand (stdio forwarding)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing forward specification after -W")
		}
		// -W is typically used with ProxyCommand
		server.ProxyCommand = fmt.Sprintf("ssh -W %s %%h:%%p", parts[i+1])
		return 2, nil
	}

	return 0, nil
}

// parseLoggingFlag parses logging and debugging flags
func parseLoggingFlag(server *domain.Server, parts []string, i int) int {
	part := parts[i]

	switch part {
	case "-v":
		// Verbose (can be repeated for more verbosity)
		switch server.LogLevel {
		case "":
			server.LogLevel = logVerbose
		case logVerbose:
			server.LogLevel = logDebug
		case logDebug:
			server.LogLevel = logDebug2
		case logDebug2:
			server.LogLevel = logDebug3
		}
		return 1

	case "-vv":
		// Double verbose
		server.LogLevel = logDebug
		return 1

	case "-vvv":
		// Triple verbose
		server.LogLevel = logDebug2
		return 1

	case "-q":
		// Quiet mode
		server.LogLevel = logQuiet
		return 1
	}

	return 0
}

// parseMiscFlag parses miscellaneous flags
func parseMiscFlag(server *domain.Server, parts []string, i int) (int, error) {
	part := parts[i]

	switch part {
	case "-c":
		// Cipher specification
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing cipher specification after -c")
		}
		server.Ciphers = parts[i+1]
		return 2, nil

	case "-m":
		// MAC specification
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing MAC specification after -m")
		}
		server.MACs = parts[i+1]
		return 2, nil

	case "-e":
		// Escape character
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing escape character after -e")
		}
		server.EscapeChar = parts[i+1]
		return 2, nil

	case "-F":
		// Config file (skip for now, as we don't process external configs)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing config file after -F")
		}
		// Skip the config file path
		return 2, nil

	case "-M":
		// ControlMaster mode
		server.ControlMaster = sshYes
		return 1, nil

	case "-S":
		// ControlPath
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing control path after -S")
		}
		server.ControlPath = parts[i+1]
		return 2, nil

	case "-O":
		// Control command (check, forward, cancel, exit, stop)
		if i+1 >= len(parts) {
			return 0, fmt.Errorf("missing control command after -O")
		}
		// Skip control command for now
		return 2, nil
	}

	return 0, nil
}

// parseSSHOption parses a single SSH -o option
func parseSSHOption(server *domain.Server, option string) error {
	// Split on the first '='
	parts := strings.SplitN(option, "=", 2)
	if len(parts) != 2 {
		// Some options use space separation
		parts = strings.SplitN(option, " ", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid option format: %s", option)
		}
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Parse different categories of options
	if err := parseConnectionOption(server, key, value); err != nil {
		return err
	}
	if err := parseAuthOption(server, key, value); err != nil {
		return err
	}
	if err := parseForwardingOption(server, key, value); err != nil {
		return err
	}
	if err := parseSecurityOption(server, key, value); err != nil {
		return err
	}
	if err := parseMiscOption(server, key, value); err != nil {
		return err
	}

	return nil
}

// parseConnectionOption parses connection-related SSH options
func parseConnectionOption(server *domain.Server, key, value string) error {
	switch strings.ToLower(key) {
	case "hostname":
		server.Host = value
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port: %s", value)
		}
		server.Port = port
	case sshUser:
		server.User = value
	case "connecttimeout":
		server.ConnectTimeout = value
	case "serveraliveinterval":
		server.ServerAliveInterval = value
	case "serveralivecountmax":
		server.ServerAliveCountMax = value
	case "tcpkeepalive":
		server.TCPKeepAlive = value
	case "compression":
		server.Compression = value
	case "addressfamily":
		server.AddressFamily = value
	case "bindaddress":
		server.BindAddress = value
	case "bindinterface":
		server.BindInterface = value
	case "connectionattempts":
		server.ConnectionAttempts = value
	case "ipqos":
		server.IPQoS = value
	case "batchmode":
		server.BatchMode = value
	case "requesttty":
		server.RequestTTY = value
	case "sessiontype":
		server.SessionType = value
	case "remotecommand":
		server.RemoteCommand = value
	case "localcommand":
		server.LocalCommand = value
	case "permitlocalcommand":
		server.PermitLocalCommand = value
	case "escapechar":
		server.EscapeChar = value
	case "loglevel":
		server.LogLevel = value
	}
	return nil
}

// parseAuthOption parses authentication-related SSH options
func parseAuthOption(server *domain.Server, key, value string) error {
	switch strings.ToLower(key) {
	case "identityfile":
		if server.IdentityFiles == nil {
			server.IdentityFiles = []string{}
		}
		server.IdentityFiles = append(server.IdentityFiles, value)
	case "passwordauthentication":
		server.PasswordAuthentication = value
	case "pubkeyauthentication":
		server.PubkeyAuthentication = value
	case "kbdinteractiveauthentication":
		server.KbdInteractiveAuthentication = value
	case "preferredauthentications":
		server.PreferredAuthentications = value
	case "identitiesonly":
		server.IdentitiesOnly = value
	case "addkeystoagent":
		server.AddKeysToAgent = value
	case "identityagent":
		server.IdentityAgent = value
	case "numberofpasswordprompts":
		server.NumberOfPasswordPrompts = value
	case "pubkeyacceptedalgorithms":
		server.PubkeyAcceptedAlgorithms = value
	case "hostbasedacceptedalgorithms":
		server.HostbasedAcceptedAlgorithms = value
	}
	return nil
}

// parseForwardingOption parses forwarding-related SSH options
func parseForwardingOption(server *domain.Server, key, value string) error {
	switch strings.ToLower(key) {
	case "localforward":
		if server.LocalForward == nil {
			server.LocalForward = []string{}
		}
		server.LocalForward = append(server.LocalForward, value)
	case "remoteforward":
		if server.RemoteForward == nil {
			server.RemoteForward = []string{}
		}
		server.RemoteForward = append(server.RemoteForward, value)
	case "dynamicforward":
		if server.DynamicForward == nil {
			server.DynamicForward = []string{}
		}
		server.DynamicForward = append(server.DynamicForward, value)
	case "forwardagent":
		server.ForwardAgent = value
	case "forwardx11":
		server.ForwardX11 = value
	case "forwardx11trusted":
		server.ForwardX11Trusted = value
	case "gatewayports":
		server.GatewayPorts = value
	case "clearallforwardings":
		server.ClearAllForwardings = value
	case "exitonforwardfailure":
		server.ExitOnForwardFailure = value
	case "proxyjump":
		server.ProxyJump = value
	case "proxycommand":
		server.ProxyCommand = value
	}
	return nil
}

// parseSecurityOption parses security-related SSH options
func parseSecurityOption(server *domain.Server, key, value string) error {
	switch strings.ToLower(key) {
	case "stricthostkeychecking":
		server.StrictHostKeyChecking = value
	case "userknownhostsfile":
		server.UserKnownHostsFile = value
	case "checkhostip":
		server.CheckHostIP = value
	case "fingerprinthash":
		server.FingerprintHash = value
	case "hostkeyalgorithms":
		server.HostKeyAlgorithms = value
	case "ciphers":
		server.Ciphers = value
	case "macs":
		server.MACs = value
	case "kexalgorithms":
		server.KexAlgorithms = value
	case "verifyhostkeydns":
		server.VerifyHostKeyDNS = value
	case "updatehostkeys":
		server.UpdateHostKeys = value
	case "hashknownhosts":
		server.HashKnownHosts = value
	case "visualhostkey":
		server.VisualHostKey = value
	}
	return nil
}

// parseMiscOption parses miscellaneous SSH options
func parseMiscOption(server *domain.Server, key, value string) error {
	switch strings.ToLower(key) {
	case "controlmaster":
		server.ControlMaster = value
	case "controlpath":
		server.ControlPath = value
	case "controlpersist":
		server.ControlPersist = value
	case "sendenv":
		if server.SendEnv == nil {
			server.SendEnv = []string{}
		}
		server.SendEnv = append(server.SendEnv, value)
	case "setenv":
		if server.SetEnv == nil {
			server.SetEnv = []string{}
		}
		server.SetEnv = append(server.SetEnv, value)
	case "canonicalizehostname":
		server.CanonicalizeHostname = value
	case "canonicaldomains":
		server.CanonicalDomains = value
	case "canonicalizefallbacklocal":
		server.CanonicalizeFallbackLocal = value
	case "canonicalizemaxdots":
		server.CanonicalizeMaxDots = value
	case "canonicalizepermittedcnames":
		server.CanonicalizePermittedCNAMEs = value
	}
	return nil
}

// preprocessMultilineCommand handles backslash line continuation
// It joins lines that end with backslash into a single line
func preprocessMultilineCommand(cmd string) string {
	// Split by newlines to handle each line
	lines := strings.Split(cmd, "\n")
	var result strings.Builder

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check if line ends with backslash (line continuation)
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "\\") {
			// Remove the trailing backslash and add the content
			content := strings.TrimSuffix(trimmed, "\\")
			result.WriteString(strings.TrimSpace(content))
			// Add a space to separate from next line's content
			if i < len(lines)-1 {
				result.WriteString(" ")
			}
		} else {
			// Normal line without continuation
			result.WriteString(strings.TrimSpace(line))
			// Only add space if there are more lines and this isn't empty
			if i < len(lines)-1 && strings.TrimSpace(line) != "" {
				result.WriteString(" ")
			}
		}
	}

	return strings.TrimSpace(result.String())
}

// splitCommand splits a command string into parts, preserving quoted strings
func splitCommand(cmd string) ([]string, error) {
	var parts []string
	var current strings.Builder
	var inQuote rune
	var escaped bool

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		if r == '"' || r == '\'' {
			inQuote = r
			continue
		}

		if r == ' ' || r == '\t' {
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote in command")
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}
