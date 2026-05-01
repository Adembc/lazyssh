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

package ssh_config_file

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

const (
	MaxBackups         = 10
	TempSuffix         = ".tmp"
	BackupSuffix       = "lazyssh.backup"
	SSHConfigPerms     = 0o600
	OriginalBackupName = "config.original.backup"
)

// filterServers filters servers based on the query string.
func (r *Repository) filterServers(servers []domain.Server, query string) []domain.Server {
	query = strings.ToLower(query)
	filtered := make([]domain.Server, 0)

	for _, server := range servers {
		if r.matchesQuery(server, query) {
			filtered = append(filtered, server)
		}
	}

	return filtered
}

// matchesQuery checks if any field of the server matches the query string.
func (r *Repository) matchesQuery(server domain.Server, query string) bool {
	fields := []string{
		strings.ToLower(server.Host),
		strings.ToLower(server.User),
	}
	for _, tag := range server.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	for _, alias := range server.Aliases {
		fields = append(fields, strings.ToLower(alias))
	}

	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

// hostMatch records a single occurrence of an alias somewhere in the loaded
// config tree. allMatches > 1 means the alias is duplicated across files.
type hostMatch struct {
	path string
	cfg  *ssh_config.Config
	host *ssh_config.Host
}

// serverExists checks if a server with the given alias already exists anywhere
// in the loaded config (main or any included file).
func (r *Repository) serverExists(lc *loadedConfig, alias string) bool {
	matches := r.findHostMatches(lc, alias)
	return len(matches) > 0
}

// findHostMatches returns every file in lc that defines the alias, in OpenSSH
// precedence order (main first, then includes depth-first).
func (r *Repository) findHostMatches(lc *loadedConfig, alias string) []hostMatch {
	var out []hostMatch
	for i := range lc.files {
		cf := &lc.files[i]
		for _, host := range cf.cfg.Hosts {
			if r.hostContainsPattern(host, alias) {
				out = append(out, hostMatch{path: cf.path, cfg: cf.cfg, host: host})
				break
			}
		}
	}
	return out
}

// matchPaths returns just the file paths from a slice of hostMatches.
func matchPaths(ms []hostMatch) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.path)
	}
	return out
}

// hostContainsPattern checks if a host contains a specific pattern.
func (r *Repository) hostContainsPattern(host *ssh_config.Host, target string) bool {
	for _, pattern := range host.Patterns {
		if pattern.String() == target {
			return true
		}
	}
	return false
}

// createHostFromServer creates a new ssh_config.Host from a domain.Server.
func (r *Repository) createHostFromServer(server domain.Server) *ssh_config.Host {
	host := &ssh_config.Host{
		Patterns: []*ssh_config.Pattern{
			{Str: server.Alias},
		},
		Nodes:              make([]ssh_config.Node, 0),
		EOLComment:         "Added by lazyssh",
		SpaceBeforeComment: strings.Repeat(" ", 4),
	}

	// Basic config - always present
	r.addKVNodeIfNotEmpty(host, "HostName", server.Host)
	r.addKVNodeIfNotEmpty(host, "User", server.User)
	if server.Port != 0 {
		r.addKVNodeIfNotEmpty(host, "Port", fmt.Sprintf("%d", server.Port))
	}
	for _, identityFile := range server.IdentityFiles {
		r.addKVNodeIfNotEmpty(host, "IdentityFile", identityFile)
	}

	// Connection and proxy settings
	r.addKVNodeIfNotEmpty(host, "ProxyJump", server.ProxyJump)
	r.addKVNodeIfNotEmpty(host, "ProxyCommand", server.ProxyCommand)
	r.addKVNodeIfNotEmpty(host, "RemoteCommand", server.RemoteCommand)
	r.addKVNodeIfNotEmpty(host, "RequestTTY", server.RequestTTY)
	r.addKVNodeIfNotEmpty(host, "ConnectTimeout", server.ConnectTimeout)
	r.addKVNodeIfNotEmpty(host, "ConnectionAttempts", server.ConnectionAttempts)

	// Port forwarding
	for _, forward := range server.LocalForward {
		configFormat := r.convertCLIForwardToConfigFormat(forward)
		r.addKVNodeIfNotEmpty(host, "LocalForward", configFormat)
	}
	for _, forward := range server.RemoteForward {
		configFormat := r.convertCLIForwardToConfigFormat(forward)
		r.addKVNodeIfNotEmpty(host, "RemoteForward", configFormat)
	}
	for _, forward := range server.DynamicForward {
		r.addKVNodeIfNotEmpty(host, "DynamicForward", forward)
	}

	// Authentication and key management
	r.addKVNodeIfNotEmpty(host, "PubkeyAuthentication", server.PubkeyAuthentication)
	r.addKVNodeIfNotEmpty(host, "PubkeyAcceptedAlgorithms", server.PubkeyAcceptedAlgorithms)
	r.addKVNodeIfNotEmpty(host, "HostbasedAcceptedAlgorithms", server.HostbasedAcceptedAlgorithms)
	r.addKVNodeIfNotEmpty(host, "PasswordAuthentication", server.PasswordAuthentication)
	r.addKVNodeIfNotEmpty(host, "PreferredAuthentications", server.PreferredAuthentications)
	r.addKVNodeIfNotEmpty(host, "IdentitiesOnly", server.IdentitiesOnly)
	r.addKVNodeIfNotEmpty(host, "AddKeysToAgent", server.AddKeysToAgent)
	r.addKVNodeIfNotEmpty(host, "IdentityAgent", server.IdentityAgent)

	// Agent and X11 forwarding
	r.addKVNodeIfNotEmpty(host, "ForwardAgent", server.ForwardAgent)
	r.addKVNodeIfNotEmpty(host, "ForwardX11", server.ForwardX11)
	r.addKVNodeIfNotEmpty(host, "ForwardX11Trusted", server.ForwardX11Trusted)

	// Connection multiplexing
	r.addKVNodeIfNotEmpty(host, "ControlMaster", server.ControlMaster)
	r.addKVNodeIfNotEmpty(host, "ControlPath", server.ControlPath)
	r.addKVNodeIfNotEmpty(host, "ControlPersist", server.ControlPersist)

	// Connection reliability
	r.addKVNodeIfNotEmpty(host, "ServerAliveInterval", server.ServerAliveInterval)
	r.addKVNodeIfNotEmpty(host, "ServerAliveCountMax", server.ServerAliveCountMax)
	r.addKVNodeIfNotEmpty(host, "Compression", server.Compression)
	r.addKVNodeIfNotEmpty(host, "TCPKeepAlive", server.TCPKeepAlive)
	r.addKVNodeIfNotEmpty(host, "BatchMode", server.BatchMode)

	// Security
	r.addKVNodeIfNotEmpty(host, "StrictHostKeyChecking", server.StrictHostKeyChecking)
	r.addKVNodeIfNotEmpty(host, "UserKnownHostsFile", server.UserKnownHostsFile)
	r.addKVNodeIfNotEmpty(host, "HostKeyAlgorithms", server.HostKeyAlgorithms)
	r.addKVNodeIfNotEmpty(host, "VerifyHostKeyDNS", server.VerifyHostKeyDNS)
	r.addKVNodeIfNotEmpty(host, "UpdateHostKeys", server.UpdateHostKeys)
	r.addKVNodeIfNotEmpty(host, "HashKnownHosts", server.HashKnownHosts)
	r.addKVNodeIfNotEmpty(host, "VisualHostKey", server.VisualHostKey)

	// Command execution
	r.addKVNodeIfNotEmpty(host, "LocalCommand", server.LocalCommand)
	r.addKVNodeIfNotEmpty(host, "PermitLocalCommand", server.PermitLocalCommand)
	r.addKVNodeIfNotEmpty(host, "EscapeChar", server.EscapeChar)

	// Environment settings
	for _, env := range server.SendEnv {
		r.addKVNodeIfNotEmpty(host, "SendEnv", env)
	}
	for _, env := range server.SetEnv {
		r.addKVNodeIfNotEmpty(host, "SetEnv", env)
	}

	// Debugging
	r.addKVNodeIfNotEmpty(host, "LogLevel", server.LogLevel)

	return host
}

// addKVNodeIfNotEmpty adds a key-value node to the host if the value is not empty.
func (r *Repository) addKVNodeIfNotEmpty(host *ssh_config.Host, key, value string) {
	if value == "" {
		return
	}

	kvNode := &ssh_config.KV{
		Key:          key,
		Value:        value,
		LeadingSpace: 4,
	}
	r.insertKVNodeAfterLastKV(host, kvNode)
}

// insertKVNodeAfterLastKV inserts a KV node immediately after the last existing KV node in the host.
// This preserves any trailing non-KV nodes (blank lines, comments) that may exist after the host's
// configuration block, preventing formatting shifts when adding new fields to a host entry.
//
// Example: If a host block has trailing blank lines separating it from the next host entry,
// this function ensures new fields are inserted before those blank lines, maintaining the
// visual separation between host blocks.
func (r *Repository) insertKVNodeAfterLastKV(host *ssh_config.Host, kvNode *ssh_config.KV) {
	// STEP 1: Find the last KV node (search backwards)
	lastKVIndex := -1
	for i := len(host.Nodes) - 1; i >= 0; i-- {
		if _, ok := host.Nodes[i].(*ssh_config.KV); ok {
			lastKVIndex = i
			break
		}
	}

	// STEP 2: Handle case where no KV nodes exist
	if lastKVIndex == -1 {
		if len(host.Nodes) == 0 {
			// Case A: Empty host - just append
			host.Nodes = append(host.Nodes, kvNode)
		} else {
			// Case B: Only comments/blanks exist - prepend before them
			host.Nodes = append([]ssh_config.Node{kvNode}, host.Nodes...)
		}
		return
	}

	// STEP 3: We found KV nodes - insert after the last one
	insertAt := lastKVIndex + 1

	if insertAt == len(host.Nodes) {
		// Case C: Last KV is at the end - just append
		host.Nodes = append(host.Nodes, kvNode)
		return
	}

	// Case D: Last KV has trailing nodes (blanks/comments) - insert between them
	host.Nodes = append(host.Nodes[:insertAt], append([]ssh_config.Node{kvNode}, host.Nodes[insertAt:]...)...)
}

// removeNodesByKey removes all nodes with the specified key from the nodes slice
func removeNodesByKey(nodes []ssh_config.Node, key string) []ssh_config.Node {
	filtered := make([]ssh_config.Node, 0, len(nodes))
	for _, node := range nodes {
		if kv, ok := node.(*ssh_config.KV); ok {
			if strings.EqualFold(kv.Key, key) {
				continue // skip nodes with matching key
			}
		}
		filtered = append(filtered, node)
	}
	return filtered
}

// scalarFieldMap returns the canonical key→value map for a server's scalar
// SSH config fields. Used by updateHostNodes to diff old vs new and apply
// only the keys that actually changed (so editing a host in one of several
// Include files doesn't pollute it with the merged view's other fields).
func scalarFieldMap(s domain.Server) map[string]string {
	portValue := ""
	if s.Port != 0 {
		portValue = fmt.Sprintf("%d", s.Port)
	}
	return map[string]string{
		"hostname":                        s.Host,
		"user":                            s.User,
		"port":                            portValue,
		"proxycommand":                    s.ProxyCommand,
		"proxyjump":                       s.ProxyJump,
		"remotecommand":                   s.RemoteCommand,
		"requesttty":                      s.RequestTTY,
		"sessiontype":                     s.SessionType,
		"connecttimeout":                  s.ConnectTimeout,
		"connectionattempts":              s.ConnectionAttempts,
		"bindaddress":                     s.BindAddress,
		"bindinterface":                   s.BindInterface,
		"addressfamily":                   s.AddressFamily,
		"exitonforwardfailure":            s.ExitOnForwardFailure,
		"ipqos":                           s.IPQoS,
		"canonicalizehostname":            s.CanonicalizeHostname,
		"canonicaldomains":                s.CanonicalDomains,
		"canonicalizefallbacklocal":       s.CanonicalizeFallbackLocal,
		"canonicalizemaxdots":             s.CanonicalizeMaxDots,
		"canonicalizepermittedcnames":     s.CanonicalizePermittedCNAMEs,
		"clearallforwardings":             s.ClearAllForwardings,
		"gatewayports":                    s.GatewayPorts,
		"pubkeyauthentication":            s.PubkeyAuthentication,
		"passwordauthentication":          s.PasswordAuthentication,
		"preferredauthentications":        s.PreferredAuthentications,
		"pubkeyacceptedalgorithms":        s.PubkeyAcceptedAlgorithms,
		"pubkeyacceptedkeytypes":          s.PubkeyAcceptedAlgorithms,
		"hostbasedacceptedalgorithms":     s.HostbasedAcceptedAlgorithms,
		"hostbasedkeytypes":               s.HostbasedAcceptedAlgorithms,
		"hostbasedacceptedkeytypes":       s.HostbasedAcceptedAlgorithms,
		"identitiesonly":                  s.IdentitiesOnly,
		"addkeystoagent":                  s.AddKeysToAgent,
		"identityagent":                   s.IdentityAgent,
		"kbdinteractiveauthentication":    s.KbdInteractiveAuthentication,
		"challengeresponseauthentication": s.KbdInteractiveAuthentication,
		"numberofpasswordprompts":         s.NumberOfPasswordPrompts,
		"forwardagent":                    s.ForwardAgent,
		"forwardx11":                      s.ForwardX11,
		"forwardx11trusted":               s.ForwardX11Trusted,
		"controlmaster":                   s.ControlMaster,
		"controlpath":                     s.ControlPath,
		"controlpersist":                  s.ControlPersist,
		"serveraliveinterval":             s.ServerAliveInterval,
		"serveralivecountmax":             s.ServerAliveCountMax,
		"compression":                     s.Compression,
		"tcpkeepalive":                    s.TCPKeepAlive,
		"batchmode":                       s.BatchMode,
		"stricthostkeychecking":           s.StrictHostKeyChecking,
		"checkhostip":                     s.CheckHostIP,
		"fingerprinthash":                 s.FingerprintHash,
		"userknownhostsfile":              s.UserKnownHostsFile,
		"hostkeyalgorithms":               s.HostKeyAlgorithms,
		"macs":                            s.MACs,
		"ciphers":                         s.Ciphers,
		"kexalgorithms":                   s.KexAlgorithms,
		"verifyhostkeydns":                s.VerifyHostKeyDNS,
		"updatehostkeys":                  s.UpdateHostKeys,
		"hashknownhosts":                  s.HashKnownHosts,
		"visualhostkey":                   s.VisualHostKey,
		"localcommand":                    s.LocalCommand,
		"permitlocalcommand":              s.PermitLocalCommand,
		"escapechar":                      s.EscapeChar,
		"loglevel":                        s.LogLevel,
	}
}

// updateHostNodes applies the diff between oldServer and newServer to host's
// KV nodes. Unchanged fields are not touched, so editing a single field on a
// host that's defined across multiple Include files won't drag the merged
// view's other fields into the file being written.
func (r *Repository) updateHostNodes(host *ssh_config.Host, oldServer, newServer domain.Server) {
	oldVals := scalarFieldMap(oldServer)
	newVals := scalarFieldMap(newServer)
	for key, newVal := range newVals {
		if oldVals[key] == newVal {
			continue
		}
		if newVal != "" {
			r.updateOrAddKVNode(host, key, newVal)
		} else {
			r.removeKVNode(host, key)
		}
	}

	r.updateListField(host, "IdentityFile", oldServer.IdentityFiles, newServer.IdentityFiles, nil)
	r.updateListField(host, "LocalForward", oldServer.LocalForward, newServer.LocalForward, r.convertCLIForwardToConfigFormat)
	r.updateListField(host, "RemoteForward", oldServer.RemoteForward, newServer.RemoteForward, r.convertCLIForwardToConfigFormat)
	r.updateListField(host, "DynamicForward", oldServer.DynamicForward, newServer.DynamicForward, nil)
	r.updateListField(host, "SendEnv", oldServer.SendEnv, newServer.SendEnv, nil)
	r.updateListField(host, "SetEnv", oldServer.SetEnv, newServer.SetEnv, nil)
}

// updateListField rewrites a multi-valued KV (e.g. IdentityFile) only when
// its values actually changed. transform is applied per-value before writing
// (used for converting CLI forwarding format to SSH config format); pass nil
// for an identity transform.
func (r *Repository) updateListField(host *ssh_config.Host, key string, oldVals, newVals []string, transform func(string) string) {
	if slices.Equal(oldVals, newVals) {
		return
	}
	host.Nodes = removeNodesByKey(host.Nodes, key)
	for _, v := range newVals {
		if transform != nil {
			v = transform(v)
		}
		r.addKVNodeIfNotEmpty(host, key, v)
	}
}

// updateOrAddKVNode updates an existing key-value node or adds a new one if it doesn't exist.
func (r *Repository) updateOrAddKVNode(host *ssh_config.Host, key, newValue string) {
	// Try to update existing node
	for _, node := range host.Nodes {
		kvNode, ok := node.(*ssh_config.KV)
		if ok && strings.EqualFold(kvNode.Key, key) {
			kvNode.Value = newValue
			return
		}
	}

	// Add new node if not found
	kvNode := &ssh_config.KV{
		Key:          r.getProperKeyCase(key),
		Value:        newValue,
		LeadingSpace: 4,
	}
	r.insertKVNodeAfterLastKV(host, kvNode)
}

// removeKVNode removes a key-value node from the host if it exists.
func (r *Repository) removeKVNode(host *ssh_config.Host, key string) {
	filtered := make([]ssh_config.Node, 0, len(host.Nodes))
	for _, node := range host.Nodes {
		if kvNode, ok := node.(*ssh_config.KV); ok {
			if strings.EqualFold(kvNode.Key, key) {
				continue // Skip this node (remove it)
			}
		}
		filtered = append(filtered, node)
	}
	host.Nodes = filtered
}

// getProperKeyCase returns the proper case for known SSH config keys.
// Reference: https://www.ssh.com/academy/ssh/config
func (r *Repository) getProperKeyCase(key string) string {
	keyMap := map[string]string{
		"hostname":                        "HostName",
		"user":                            "User",
		"port":                            "Port",
		"identityfile":                    "IdentityFile",
		"proxycommand":                    "ProxyCommand",
		"proxyjump":                       "ProxyJump",
		"remotecommand":                   "RemoteCommand",
		"requesttty":                      "RequestTTY",
		"sessiontype":                     "SessionType",
		"connecttimeout":                  "ConnectTimeout",
		"connectionattempts":              "ConnectionAttempts",
		"bindaddress":                     "BindAddress",
		"bindinterface":                   "BindInterface",
		"addressfamily":                   "AddressFamily",
		"exitonforwardfailure":            "ExitOnForwardFailure",
		"ipqos":                           "IPQoS",
		"canonicalizehostname":            "CanonicalizeHostname",
		"canonicaldomains":                "CanonicalDomains",
		"canonicalizefallbacklocal":       "CanonicalizeFallbackLocal",
		"canonicalizemaxdots":             "CanonicalizeMaxDots",
		"canonicalizepermittedcnames":     "CanonicalizePermittedCNAMEs",
		"localforward":                    "LocalForward",
		"remoteforward":                   "RemoteForward",
		"dynamicforward":                  "DynamicForward",
		"clearallforwardings":             "ClearAllForwardings",
		"gatewayports":                    "GatewayPorts",
		"pubkeyauthentication":            "PubkeyAuthentication",
		"passwordauthentication":          "PasswordAuthentication",
		"preferredauthentications":        "PreferredAuthentications",
		"pubkeyacceptedalgorithms":        "PubkeyAcceptedAlgorithms",
		"pubkeyacceptedkeytypes":          "PubkeyAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"hostbasedacceptedalgorithms":     "HostbasedAcceptedAlgorithms",
		"hostbasedkeytypes":               "HostbasedAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"hostbasedacceptedkeytypes":       "HostbasedAcceptedAlgorithms", // Deprecated alias (since OpenSSH 8.5)
		"identitiesonly":                  "IdentitiesOnly",
		"addkeystoagent":                  "AddKeysToAgent",
		"identityagent":                   "IdentityAgent",
		"kbdinteractiveauthentication":    "KbdInteractiveAuthentication",
		"challengeresponseauthentication": "KbdInteractiveAuthentication", // Deprecated alias
		"numberofpasswordprompts":         "NumberOfPasswordPrompts",
		"forwardagent":                    "ForwardAgent",
		"forwardx11":                      "ForwardX11",
		"forwardx11trusted":               "ForwardX11Trusted",
		"controlmaster":                   "ControlMaster",
		"controlpath":                     "ControlPath",
		"controlpersist":                  "ControlPersist",
		"serveraliveinterval":             "ServerAliveInterval",
		"serveralivecountmax":             "ServerAliveCountMax",
		"compression":                     "Compression",
		"tcpkeepalive":                    "TCPKeepAlive",
		"stricthostkeychecking":           "StrictHostKeyChecking",
		"checkhostip":                     "CheckHostIP",
		"fingerprinthash":                 "FingerprintHash",
		"verifyhostkeydns":                "VerifyHostKeyDNS",
		"updatehostkeys":                  "UpdateHostKeys",
		"hashknownhosts":                  "HashKnownHosts",
		"visualhostkey":                   "VisualHostKey",
		"userknownhostsfile":              "UserKnownHostsFile",
		"hostkeyalgorithms":               "HostKeyAlgorithms",
		"macs":                            "MACs",
		"ciphers":                         "Ciphers",
		"kexalgorithms":                   "KexAlgorithms",
		"localcommand":                    "LocalCommand",
		"permitlocalcommand":              "PermitLocalCommand",
		"escapechar":                      "EscapeChar",
		"sendenv":                         "SendEnv",
		"setenv":                          "SetEnv",
		"loglevel":                        "LogLevel",
		"batchmode":                       "BatchMode",
	}

	if properCase, exists := keyMap[strings.ToLower(key)]; exists {
		return properCase
	}
	return key
}

// convertCLIForwardToConfigFormat converts CLI format forwarding spec to SSH config format.
// CLI format: [bind_address:]port:host:hostport
// Config format: [bind_address:]port host:hostport
func (r *Repository) convertCLIForwardToConfigFormat(forward string) string {
	// Handle IPv6 addresses in brackets like [2001:db8::1]
	// These should be treated as a single unit

	// Find the last `:digits` that represents the final port
	lastPortStart := -1
	for i := len(forward) - 1; i >= 0; i-- {
		if forward[i] == ':' {
			// Check if everything after this colon is digits
			if i+1 < len(forward) {
				allDigits := true
				hasDigits := false
				for j := i + 1; j < len(forward); j++ {
					if forward[j] >= '0' && forward[j] <= '9' {
						hasDigits = true
					} else {
						allDigits = false
						break
					}
				}
				if allDigits && hasDigits {
					lastPortStart = i
					break
				}
			}
		}
	}

	if lastPortStart == -1 {
		// No port at the end, return as-is
		return forward
	}

	// Now find the split point between local and remote parts
	// We need to handle bracket-enclosed addresses specially
	inBrackets := 0
	for i := lastPortStart - 1; i >= 0; i-- {
		switch forward[i] {
		case ']':
			inBrackets++
		case '[':
			inBrackets--
		case ':':
			if inBrackets != 0 {
				continue
			}
			// This colon is not inside brackets
			// Check if this looks like it could be the split point
			// The split point would be after a port number (digits after a colon)

			// Look ahead to see what comes after this colon
			nextChar := byte(' ')
			if i+1 < len(forward) {
				nextChar = forward[i+1]
			}

			// If the next character could be start of a host (letter, digit, bracket)
			// then this is our split point
			if nextChar != ':' {
				localPart := forward[:i]
				remotePart := forward[i+1:]
				return localPart + " " + remotePart
			}
		}
	}

	// If no split point found, return as-is
	return forward
}

// removeHostByAlias removes a host by its alias from the list of hosts.
func (r *Repository) removeHostByAlias(hosts []*ssh_config.Host, alias string) []*ssh_config.Host {
	for i, host := range hosts {
		if r.hostContainsPattern(host, alias) {
			return append(hosts[:i], hosts[i+1:]...)
		}
	}
	return hosts
}

// preferenceResolves reports whether preferPath unambiguously selects one of
// the matches. Empty preferPath never resolves; an unknown path also doesn't.
func preferenceResolves(matches []hostMatch, preferPath string) bool {
	if preferPath == "" {
		return false
	}
	for _, m := range matches {
		if m.path == preferPath {
			return true
		}
	}
	return false
}

// pickWritableMatch chooses which match to mutate when callers haven't passed
// a preferred file. If preferPath is non-empty and matches one of the
// candidates, we use that. Otherwise the first (highest-precedence) match
// wins. Callers must ensure matches is non-empty.
func pickWritableMatch(matches []hostMatch, preferPath string) hostMatch {
	if preferPath != "" {
		for _, m := range matches {
			if m.path == preferPath {
				return m
			}
		}
	}
	return matches[0]
}
