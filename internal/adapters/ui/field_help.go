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

	"github.com/Adembc/lazyssh/internal/i18n"
)

// FieldHelp contains help information for SSH config fields
type FieldHelp struct {
	Field       string   // Field name
	Description string   // Brief description
	Syntax      string   // Syntax format
	Examples    []string // Usage examples
	Default     string   // Default value
	Since       string   // OpenSSH version when introduced
	Category    string   // Category for grouping
}

// HelpDisplayMode defines how help is displayed
type HelpDisplayMode int

const (
	HelpModeOff     HelpDisplayMode = iota // No help shown
	HelpModeCompact                        // Single line help
	HelpModeNormal                         // Standard help panel
	HelpModeFull                           // Detailed help with all info
)

// translateOr returns the translation for key if found, otherwise returns fallback.
func translateOr(key, fallback string) string {
	if msg := i18n.T(key); msg != key {
		return msg
	}
	return fallback
}

// GetFieldHelp returns help information for a specific field
func GetFieldHelp(fieldName string) *FieldHelp {
	if help, exists := fieldHelpData[fieldName]; exists {
		// Update default value from centralized source
		defaultValue := GetSSHFieldDefault(fieldName)
		if defaultValue != "" {
			help.Default = formatDefaultValue(fieldName, defaultValue)
		}
		// Translate user-facing text
		fieldKey := "field_help." + strings.ToLower(fieldName)
		help.Description = translateOr(fieldKey, help.Description)
		help.Syntax = translateOr(fieldKey+".syntax", help.Syntax)
		help.Default = translateOr(fieldKey+".default", help.Default)
		help.Since = translateOr(fieldKey+".since", help.Since)
		if len(help.Examples) > 0 {
			examplesKey := fieldKey + ".examples"
			if exStr := i18n.T(examplesKey); exStr != examplesKey {
				help.Examples = strings.Split(exStr, "|")
			}
		}
		return &help
	}
	return nil
}

// formatDefaultValue formats the default value for display in help
func formatDefaultValue(fieldName, value string) string {
	// Special formatting for certain fields
	switch fieldName {
	case "ConnectTimeout":
		if value == "" {
			return i18n.T("default.none_system")
		}
		return fmt.Sprintf(i18n.T("default.seconds"), value)
	case "ServerAliveInterval":
		if value == "0" {
			return i18n.T("default.disabled")
		}
		return fmt.Sprintf(i18n.T("default.seconds"), value)
	case "ControlPath", "ProxyJump", "ProxyCommand", "RemoteCommand",
		"LocalForward", "RemoteForward", "DynamicForward",
		"LocalCommand", "SendEnv", "SetEnv", "BindAddress", "BindInterface",
		"CanonicalDomains", "CanonicalizePermittedCNAMEs",
		"PubkeyAcceptedAlgorithms", "HostbasedAcceptedAlgorithms",
		"HostKeyAlgorithms", "Ciphers", "MACs", "KexAlgorithms":
		if value == "" {
			return i18n.T("default.none")
		}
		return value
	case "PreferredAuthentications":
		if value == "gssapi-with-mic,hostbased,publickey,keyboard-interactive,password" {
			return "gssapi-with-mic,hostbased,publickey,keyboard-interactive,password"
		}
		return value
	case "IdentityAgent":
		if value == "SSH_AUTH_SOCK" {
			return "SSH_AUTH_SOCK"
		}
		return value
	case "User":
		if value == "" {
			return i18n.T("default.current_username")
		}
		return value
	default:
		return value
	}
}

// fieldHelpData contains help information for all SSH config fields
var fieldHelpData = map[string]FieldHelp{
	// Basic fields
	"Alias": {
		Field:       "Alias",
		Description: "A nickname or abbreviation for the host. This is what you type after 'ssh' command.",
		Syntax:      "any_string_without_spaces",
		Examples:    []string{"myserver", "prod-db", "dev-web-01"},
		Default:     "(required)",
		Category:    "Basic",
	},
	"Host": {
		Field:       "Host",
		Description: "The real hostname or IP address to connect to. Can be a domain name or IP address.",
		Syntax:      "hostname | ip_address",
		Examples:    []string{"example.com", "192.168.1.100", "2001:db8::1"},
		Default:     "(required)",
		Category:    "Basic",
	},
	"Port": {
		Field:       "Port",
		Description: "The port number to connect to on the remote host. Standard SSH port is 22.",
		Syntax:      "port_number (1-65535)",
		Examples:    []string{"22", "2222", "8022"},
		Default:     "22",
		Category:    "Basic",
	},
	"User": {
		Field:       "User",
		Description: "Username for logging into the remote machine. If not specified, uses current username.",
		Syntax:      "username",
		Examples:    []string{"root", "ubuntu", "admin", "deploy"},
		Default:     "current username",
		Category:    "Basic",
	},
	"Keys": {
		Field:       "Keys",
		Description: "Path to SSH private key files for authentication. Multiple keys can be specified.",
		Syntax:      "path[,path,...]",
		Examples:    []string{"~/.ssh/id_ed25519", "~/.ssh/id_rsa,~/.ssh/id_ed25519"},
		Default:     "~/.ssh/id_rsa, ~/.ssh/id_ed25519, etc.",
		Category:    "Basic",
	},
	"AuthMethod": {
		Field:       "AuthMethod",
		Description: "选择认证方式：密钥、密码或两者同时使用。",
		Syntax:      "key | password | key_password",
		Examples:    []string{"key", "password"},
		Default:     "key",
		Category:    "Basic",
	},
	"LoginMode": {
		Field:       "LoginMode",
		Description: "登录模式：交互式始终请求 TTY，批处理不请求 TTY，自动模式根据需要请求。",
		Syntax:      "interactive | batch | auto",
		Examples:    []string{"interactive", "batch", "auto"},
		Default:     "interactive",
		Category:    "Basic",
	},

	// Connection fields
	"ProxyJump": {
		Field:       "ProxyJump",
		Description: "Specifies one or more jump hosts (bastion hosts) to reach the destination. Useful for accessing servers behind firewalls.",
		Syntax:      "[user@]host[:port][,[user@]host[:port]]",
		Examples:    []string{"bastion.example.com", "jump1.com,jump2.com", "user@proxy:2222"},
		Default:     "none",
		Since:       "OpenSSH 7.3+",
		Category:    "Connection",
	},
	"ProxyCommand": {
		Field:       "ProxyCommand",
		Description: "Command to use to connect to the server. Useful for connecting through proxies or using custom connection methods.",
		Syntax:      "command",
		Examples:    []string{"ssh -W %h:%p jump.example.com", "nc -X 5 -x proxy:1080 %h %p"},
		Default:     "none",
		Category:    "Connection",
	},
	"RemoteCommand": {
		Field:       "RemoteCommand",
		Description: "Specifies a command to execute on the remote machine after successfully connecting.",
		Syntax:      "command | none",
		Examples:    []string{"tmux attach || tmux new", "screen -r", "none"},
		Default:     "none",
		Since:       "OpenSSH 7.6+ (for 'none' value)",
		Category:    "Connection",
	},
	"ConnectTimeout": {
		Field:       "ConnectTimeout",
		Description: "Timeout in seconds for establishing the connection. Useful for slow or unreliable networks.",
		Syntax:      "seconds | none",
		Examples:    []string{"10", "30", "none"},
		Default:     "none (system default)",
		Category:    "Connection",
	},
	"ConnectionAttempts": {
		Field:       "ConnectionAttempts",
		Description: "Number of attempts to make before giving up on connecting.",
		Syntax:      "number",
		Examples:    []string{"1", "3", "5"},
		Default:     "1",
		Category:    "Connection",
	},
	"SessionType": {
		Field:       "SessionType",
		Description: "Type of session to request. 'none' (-N flag) is useful for port forwarding without shell.",
		Syntax:      "none | subsystem | default",
		Examples:    []string{"none", "subsystem", "default"},
		Default:     "default",
		Since:       "OpenSSH 8.7+",
		Category:    "Connection",
	},
	"RequestTTY": {
		Field:       "RequestTTY",
		Description: "Request a pseudo-terminal for the session. Required for interactive programs.",
		Syntax:      "yes | no | force | auto",
		Examples:    []string{"yes", "force", "auto"},
		Default:     "auto",
		Category:    "Connection",
	},

	// Port forwarding fields
	"LocalForward": {
		Field:       "LocalForward",
		Description: "Forward a local port to a remote address. Useful for accessing remote services through SSH tunnel.",
		Syntax:      "[bind_address:]port:host:hostport (CLI format, auto-converted for config file)",
		Examples:    []string{"8080:localhost:80", "3306:db.internal:3306", "*:8080:localhost:80"},
		Default:     "none",
		Category:    "Forwarding",
	},
	"RemoteForward": {
		Field:       "RemoteForward",
		Description: "Forward a remote port to a local address. Allows remote users to access local services.",
		Syntax:      "[bind_address:]port:host:hostport (CLI format, auto-converted for config file)",
		Examples:    []string{"8080:localhost:3000", "*:80:localhost:8080"},
		Default:     "none",
		Category:    "Forwarding",
	},
	"DynamicForward": {
		Field:       "DynamicForward",
		Description: "Create a SOCKS proxy on the specified port. Useful for routing traffic through SSH.",
		Syntax:      "[bind_address:]port",
		Examples:    []string{"1080", "localhost:1080", "*:1080"},
		Default:     "none",
		Category:    "Forwarding",
	},
	"ForwardAgent": {
		Field:       "ForwardAgent",
		Description: "Forward SSH agent connection to remote host. Allows using local SSH keys on remote servers.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Forwarding",
	},
	"ForwardX11": {
		Field:       "ForwardX11",
		Description: "Enable X11 forwarding for GUI applications over SSH.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Forwarding",
	},

	// Authentication fields
	"PubkeyAuthentication": {
		Field:       "PubkeyAuthentication",
		Description: "Enable or disable public key authentication.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "yes",
		Category:    "Authentication",
	},
	"PasswordAuthentication": {
		Field:       "PasswordAuthentication",
		Description: "Enable or disable password authentication.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "yes",
		Category:    "Authentication",
	},
	"PreferredAuthentications": {
		Field:       "PreferredAuthentications",
		Description: "Order of authentication methods to try.",
		Syntax:      "method[,method,...]",
		Examples:    []string{"publickey,password", "publickey,keyboard-interactive,password"},
		Default:     "gssapi-with-mic,hostbased,publickey,keyboard-interactive,password",
		Category:    "Authentication",
	},
	"IdentitiesOnly": {
		Field:       "IdentitiesOnly",
		Description: "Only use authentication identity files configured in ssh_config, ignore ssh-agent.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Authentication",
	},
	"AddKeysToAgent": {
		Field:       "AddKeysToAgent",
		Description: "Add keys to ssh-agent automatically when used.",
		Syntax:      "yes | no | ask | confirm",
		Examples:    []string{"yes", "ask", "confirm"},
		Default:     "no",
		Since:       "OpenSSH 7.2+",
		Category:    "Authentication",
	},

	// Multiplexing fields
	"ControlMaster": {
		Field:       "ControlMaster",
		Description: "Enable connection multiplexing. Reuse existing connections for speed.",
		Syntax:      "yes | no | ask | auto | autoask",
		Examples:    []string{"auto", "yes", "no"},
		Default:     "no",
		Category:    "Multiplexing",
	},
	"ControlPath": {
		Field:       "ControlPath",
		Description: "Path to control socket for connection multiplexing.",
		Syntax:      "path",
		Examples:    []string{"~/.ssh/master-%r@%h:%p", "/tmp/ssh-%r@%h:%p"},
		Default:     "none",
		Category:    "Multiplexing",
	},
	"ControlPersist": {
		Field:       "ControlPersist",
		Description: "Keep master connection open in background after initial client exits.",
		Syntax:      "yes | no | time",
		Examples:    []string{"yes", "10m", "4h", "no"},
		Default:     "no",
		Category:    "Multiplexing",
	},

	// Keep-alive fields
	"ServerAliveInterval": {
		Field:       "ServerAliveInterval",
		Description: "Seconds between keepalive messages. Prevents connection drops on idle connections.",
		Syntax:      "seconds",
		Examples:    []string{"60", "120", "300"},
		Default:     "0 (disabled)",
		Category:    "Keep-Alive",
	},
	"ServerAliveCountMax": {
		Field:       "ServerAliveCountMax",
		Description: "Number of keepalive messages before disconnecting.",
		Syntax:      "count",
		Examples:    []string{"3", "5", "10"},
		Default:     "3",
		Category:    "Keep-Alive",
	},
	"TCPKeepAlive": {
		Field:       "TCPKeepAlive",
		Description: "Send TCP keepalive messages to detect broken connections.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "yes",
		Category:    "Keep-Alive",
	},

	// Security fields
	"StrictHostKeyChecking": {
		Field:       "StrictHostKeyChecking",
		Description: "How to handle unknown host keys. 'ask' prompts user, 'no' auto-adds, 'yes' requires pre-existing key.",
		Syntax:      "yes | no | ask | accept-new",
		Examples:    []string{"ask", "accept-new", "yes"},
		Default:     "ask",
		Category:    "Security",
	},
	"UserKnownHostsFile": {
		Field:       "UserKnownHostsFile",
		Description: "File to store host keys. Can specify multiple files.",
		Syntax:      "path [path ...]",
		Examples:    []string{"~/.ssh/known_hosts", "~/.ssh/known_hosts ~/.ssh/known_hosts2"},
		Default:     "~/.ssh/known_hosts",
		Category:    "Security",
	},
	"HostKeyAlgorithms": {
		Field:       "HostKeyAlgorithms",
		Description: "Host key algorithms in order of preference. Use +/- to add/remove from defaults.",
		Syntax:      "algorithm[,algorithm,...] | +algo | -algo",
		Examples:    []string{"ssh-ed25519,ssh-rsa", "+ssh-rsa", "-ssh-dss"},
		Default:     "ssh-ed25519,ecdsa-sha2-nistp256,ssh-rsa,...",
		Category:    "Security",
	},
	"Ciphers": {
		Field:       "Ciphers",
		Description: "Encryption algorithms in order of preference.",
		Syntax:      "cipher[,cipher,...] | +cipher | -cipher",
		Examples:    []string{"aes256-gcm@openssh.com,aes256-ctr", "+aes256-cbc", "-3des-cbc"},
		Default:     "chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,...",
		Category:    "Security",
	},
	"MACs": {
		Field:       "MACs",
		Description: "Message authentication code algorithms in order of preference.",
		Syntax:      "mac[,mac,...] | +mac | -mac",
		Examples:    []string{"hmac-sha2-256,hmac-sha2-512", "+hmac-md5", "-hmac-sha1"},
		Default:     "umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,...",
		Category:    "Security",
	},

	// Other useful fields
	"LogLevel": {
		Field:       "LogLevel",
		Description: "Verbosity level for logging. Higher levels show more detail for debugging.",
		Syntax:      "QUIET | FATAL | ERROR | INFO | VERBOSE | DEBUG | DEBUG1 | DEBUG2 | DEBUG3",
		Examples:    []string{"INFO", "DEBUG", "ERROR"},
		Default:     "INFO",
		Category:    "Debugging",
	},
	"Compression": {
		Field:       "Compression",
		Description: "Enable compression to reduce bandwidth usage. Useful for slow connections.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Connection",
	},
	"BatchMode": {
		Field:       "BatchMode",
		Description: "Disable all interactive prompts. Useful for scripts and automation.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Connection",
	},

	// Missing fields - Tags
	"Tags": {
		Field:       "Tags",
		Description: "Custom tags for organizing and filtering servers. Comma-separated list.",
		Syntax:      "tag1[,tag2,...]  ",
		Examples:    []string{"production", "development,staging", "web,frontend"},
		Default:     "none",
		Category:    "Basic",
	},
	"Password": {
		Field:       "Password",
		Description: "SSH password for authentication. Stored encrypted locally.",
		Syntax:      "string",
		Examples:    []string{""},
		Default:     "(stored encrypted, not in SSH config)",
		Category:    "Basic",
	},

	// Connection - IP and Address fields
	"IPQoS": {
		Field:       "IPQoS",
		Description: "Quality of Service (QoS) / DSCP / TOS for SSH connections. Can specify different values for interactive and bulk traffic.",
		Syntax:      "dscp_value | lowdelay | throughput | reliability | af11-af43 | cs0-cs7 | ef | le",
		Examples:    []string{"af21 cs1", "lowdelay throughput", "cs2"},
		Default:     "af21 cs1",
		Category:    "Connection",
	},
	"BindAddress": {
		Field:       "BindAddress",
		Description: "Use specific source address for the connection. Useful for multi-homed hosts.",
		Syntax:      "address | hostname",
		Examples:    []string{"192.168.1.100", "localhost", "*"},
		Default:     "none",
		Category:    "Connection",
	},
	"BindInterface": {
		Field:       "BindInterface",
		Description: "Use specific network interface for the connection. Useful for routing through specific NICs.",
		Syntax:      "interface_name",
		Examples:    []string{"eth0", "en0", "wlan0"},
		Default:     "none",
		Since:       "OpenSSH 7.7+",
		Category:    "Connection",
	},
	"AddressFamily": {
		Field:       "AddressFamily",
		Description: "Limit connections to IPv4 or IPv6 addresses.",
		Syntax:      "any | inet | inet6",
		Examples:    []string{"any", "inet", "inet6"},
		Default:     "any",
		Category:    "Connection",
	},

	// Hostname Canonicalization
	"CanonicalizeHostname": {
		Field:       "CanonicalizeHostname",
		Description: "Controls whether to perform hostname canonicalization. Useful for shortening hostnames.",
		Syntax:      "yes | no | always",
		Examples:    []string{"yes", "no", "always"},
		Default:     "no",
		Since:       "OpenSSH 6.5+",
		Category:    "Connection",
	},
	"CanonicalDomains": {
		Field:       "CanonicalDomains",
		Description: "Search domains for hostname canonicalization. SSH will try appending these domains.",
		Syntax:      "domain1[,domain2,...]  ",
		Examples:    []string{"example.com", "internal.net,example.org"},
		Default:     "none",
		Since:       "OpenSSH 6.5+",
		Category:    "Connection",
	},
	"CanonicalizeFallbackLocal": {
		Field:       "CanonicalizeFallbackLocal",
		Description: "Whether to fail if canonicalization fails. If yes, uses the original hostname.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "yes",
		Since:       "OpenSSH 6.5+",
		Category:    "Connection",
	},
	"CanonicalizeMaxDots": {
		Field:       "CanonicalizeMaxDots",
		Description: "Maximum dots in hostname before disabling canonicalization.",
		Syntax:      "number",
		Examples:    []string{"1", "2", "0"},
		Default:     "1",
		Since:       "OpenSSH 6.5+",
		Category:    "Connection",
	},
	"CanonicalizePermittedCNAMEs": {
		Field:       "CanonicalizePermittedCNAMEs",
		Description: "Rules for CNAME following during canonicalization.",
		Syntax:      "source:target[,source:target,...]  ",
		Examples:    []string{"*.example.com:example.net", "*.internal:*.example.com"},
		Default:     "none",
		Since:       "OpenSSH 6.5+",
		Category:    "Connection",
	},

	"ForwardX11Trusted": {
		Field:       "ForwardX11Trusted",
		Description: "Enable trusted X11 forwarding. Less secure but more compatible.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Forwarding",
	},
	"ClearAllForwardings": {
		Field:       "ClearAllForwardings",
		Description: "Clear all port forwardings set in configuration files.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Forwarding",
	},
	"ExitOnForwardFailure": {
		Field:       "ExitOnForwardFailure",
		Description: "Terminate connection if port forwarding fails.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Forwarding",
	},
	"GatewayPorts": {
		Field:       "GatewayPorts",
		Description: "Allow remote hosts to connect to forwarded ports.",
		Syntax:      "yes | no | clientspecified",
		Examples:    []string{"no", "yes", "clientspecified"},
		Default:     "no",
		Category:    "Forwarding",
	},

	// Authentication fields
	"KbdInteractiveAuthentication": {
		Field:       "KbdInteractiveAuthentication",
		Description: "Enable keyboard-interactive authentication (e.g., for 2FA).",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "yes",
		Category:    "Authentication",
	},
	"NumberOfPasswordPrompts": {
		Field:       "NumberOfPasswordPrompts",
		Description: "Number of password prompts before giving up.",
		Syntax:      "number",
		Examples:    []string{"3", "1", "5"},
		Default:     "3",
		Category:    "Authentication",
	},
	"IdentityAgent": {
		Field:       "IdentityAgent",
		Description: "Location of the authentication agent socket.",
		Syntax:      "path | SSH_AUTH_SOCK | none",
		Examples:    []string{"SSH_AUTH_SOCK", "~/.ssh/agent.sock", "none"},
		Default:     "SSH_AUTH_SOCK",
		Since:       "OpenSSH 7.3+",
		Category:    "Authentication",
	},
	"PubkeyAcceptedAlgorithms": {
		Field:       "PubkeyAcceptedAlgorithms",
		Description: "Signature algorithms accepted for public key authentication.",
		Syntax:      "algorithm[,algorithm,...]  ",
		Examples:    []string{"ssh-ed25519,ssh-rsa", "+ssh-rsa", "-ssh-dss"},
		Default:     "(all supported)",
		Since:       "OpenSSH 8.5+ (PubkeyAcceptedKeyTypes before)",
		Category:    "Authentication",
	},
	"HostbasedAcceptedAlgorithms": {
		Field:       "HostbasedAcceptedAlgorithms",
		Description: "Signature algorithms accepted for host-based authentication.",
		Syntax:      "algorithm[,algorithm,...]  ",
		Examples:    []string{"ssh-ed25519,ssh-rsa", "+ssh-rsa", "-ssh-dss"},
		Default:     "(all supported)",
		Since:       "OpenSSH 8.5+",
		Category:    "Authentication",
	},

	// Security fields
	"CheckHostIP": {
		Field:       "CheckHostIP",
		Description: "Check the host IP address in known_hosts file.",
		Syntax:      "yes | no",
		Examples:    []string{"yes", "no"},
		Default:     "no",
		Category:    "Security",
	},
	"FingerprintHash": {
		Field:       "FingerprintHash",
		Description: "Hash algorithm for displaying key fingerprints.",
		Syntax:      "md5 | sha256",
		Examples:    []string{"sha256", "md5"},
		Default:     "sha256",
		Since:       "OpenSSH 6.8+",
		Category:    "Security",
	},
	"VerifyHostKeyDNS": {
		Field:       "VerifyHostKeyDNS",
		Description: "Verify host keys using DNS SSHFP records.",
		Syntax:      "yes | no | ask",
		Examples:    []string{"no", "ask", "yes"},
		Default:     "no",
		Category:    "Security",
	},
	"UpdateHostKeys": {
		Field:       "UpdateHostKeys",
		Description: "Update known_hosts automatically with new host keys.",
		Syntax:      "yes | no | ask",
		Examples:    []string{"no", "ask", "yes"},
		Default:     "no",
		Since:       "OpenSSH 6.8+",
		Category:    "Security",
	},
	"HashKnownHosts": {
		Field:       "HashKnownHosts",
		Description: "Hash host names and addresses in known_hosts file.",
		Syntax:      "yes | no",
		Examples:    []string{"no", "yes"},
		Default:     "no",
		Category:    "Security",
	},
	"VisualHostKey": {
		Field:       "VisualHostKey",
		Description: "Display ASCII art representation of the host key.",
		Syntax:      "yes | no",
		Examples:    []string{"no", "yes"},
		Default:     "no",
		Category:    "Security",
	},

	// Cryptography
	"KexAlgorithms": {
		Field:       "KexAlgorithms",
		Description: "Key exchange algorithms to use.",
		Syntax:      "algorithm[,algorithm,...]  ",
		Examples:    []string{"curve25519-sha256", "+diffie-hellman-group14-sha256", "-ecdh-sha2-nistp256"},
		Default:     "(all supported)",
		Category:    "Cryptography",
	},

	// Command execution
	"LocalCommand": {
		Field:       "LocalCommand",
		Description: "Command to execute on local machine after connecting.",
		Syntax:      "command",
		Examples:    []string{"echo 'Connected to %h'", "notify-send 'SSH Connected'"},
		Default:     "none",
		Category:    "Command",
	},
	"PermitLocalCommand": {
		Field:       "PermitLocalCommand",
		Description: "Allow LocalCommand execution.",
		Syntax:      "yes | no",
		Examples:    []string{"no", "yes"},
		Default:     "no",
		Category:    "Command",
	},
	"EscapeChar": {
		Field:       "EscapeChar",
		Description: "Escape character for SSH session (~ by default). Set to 'none' to disable.",
		Syntax:      "char | none | ^char",
		Examples:    []string{"~", "^", "none"},
		Default:     "~",
		Category:    "Command",
	},

	// Environment
	"SendEnv": {
		Field:       "SendEnv",
		Description: "Environment variables to send to the server.",
		Syntax:      "variable[,variable,...]  ",
		Examples:    []string{"LANG", "LC_*", "TERM", "LANG LC_* EDITOR"},
		Default:     "none",
		Category:    "Environment",
	},
	"SetEnv": {
		Field:       "SetEnv",
		Description: "Set environment variables for the SSH session.",
		Syntax:      "VAR=value[,VAR=value,...]  ",
		Examples:    []string{"FOO=bar", "DEBUG=1", "PATH=/custom/path:$PATH"},
		Default:     "none",
		Since:       "OpenSSH 7.8+",
		Category:    "Environment",
	},
}

// GetFieldsByCategory returns all fields in a specific category
func GetFieldsByCategory(category string) []string {
	// Pre-count to allocate correct capacity
	count := 0
	for _, help := range fieldHelpData {
		if help.Category == category {
			count++
		}
	}

	fields := make([]string, 0, count)
	for name, help := range fieldHelpData {
		if help.Category == category {
			fields = append(fields, name)
		}
	}
	return fields
}

// GetAllCategories returns all available help categories
func GetAllCategories() []string {
	categories := make(map[string]bool)
	for _, help := range fieldHelpData {
		categories[help.Category] = true
	}

	// Convert to slice with defined order
	orderedCategories := []string{
		"Basic", "Connection", "Forwarding", "Authentication",
		"Multiplexing", "Keep-Alive", "Security", "Debugging",
	}

	var result []string
	for _, cat := range orderedCategories {
		if categories[cat] {
			result = append(result, cat)
		}
	}
	return result
}
