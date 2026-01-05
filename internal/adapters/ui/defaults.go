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

// SSHFieldDefaults contains the default values for all SSH configuration fields
// This centralizes all default values to ensure consistency across the application
var SSHFieldDefaults = map[string]string{
	// Basic fields
	"Port": "22",
	"User": "", // Empty means current username (OpenSSH default)

	// Connection fields
	"ConnectTimeout":     "", // none (system default)
	"ConnectionAttempts": "1",
	"IPQoS":              "af21 cs1",
	"BatchMode":          "no",
	"Compression":        "no",
	"AddressFamily":      "any",
	"RequestTTY":         "auto",
	"SessionType":        "default",

	// Proxy fields
	"ProxyJump":     "", // none
	"ProxyCommand":  "", // none
	"RemoteCommand": "", // none

	// Port forwarding fields
	"LocalForward":         "", // none
	"RemoteForward":        "", // none
	"DynamicForward":       "", // none
	"ForwardAgent":         "no",
	"ForwardX11":           "no",
	"ForwardX11Trusted":    "no",
	"ClearAllForwardings":  "no",
	"ExitOnForwardFailure": "no",
	"GatewayPorts":         "no",

	// Authentication fields
	"PubkeyAuthentication":         "yes",
	"PasswordAuthentication":       "yes",
	"PreferredAuthentications":     "gssapi-with-mic,hostbased,publickey,keyboard-interactive,password",
	"IdentitiesOnly":               "no",
	"AddKeysToAgent":               "no",
	"IdentityAgent":                "SSH_AUTH_SOCK",
	"KbdInteractiveAuthentication": "yes",
	"NumberOfPasswordPrompts":      "3",
	"PubkeyAcceptedAlgorithms":     "", // all supported
	"HostbasedAcceptedAlgorithms":  "", // all supported

	// Multiplexing fields
	"ControlMaster":  "no",
	"ControlPath":    "", // none
	"ControlPersist": "no",

	// Keep-alive fields
	"ServerAliveInterval": "0", // disabled
	"ServerAliveCountMax": "3",
	"TCPKeepAlive":        "yes",

	// Security fields
	"StrictHostKeyChecking": "ask",
	"UserKnownHostsFile":    "~/.ssh/known_hosts",
	"HostKeyAlgorithms":     "", // default algorithms
	"Ciphers":               "", // default ciphers
	"MACs":                  "", // default MACs
	"CheckHostIP":           "no",
	"FingerprintHash":       "SHA256", // OpenSSH uses uppercase SHA256
	"VerifyHostKeyDNS":      "no",
	"UpdateHostKeys":        "no",
	"HashKnownHosts":        "no",
	"VisualHostKey":         "no",

	// Cryptography fields
	"KexAlgorithms": "", // all supported

	// Hostname canonicalization fields
	"CanonicalizeHostname":        "no",
	"CanonicalDomains":            "", // none
	"CanonicalizeFallbackLocal":   "yes",
	"CanonicalizeMaxDots":         "1",
	"CanonicalizePermittedCNAMEs": "", // none

	// Command execution fields
	"LocalCommand":       "", // none
	"PermitLocalCommand": "no",
	"EscapeChar":         "~",

	// Environment fields
	"SendEnv": "", // none
	"SetEnv":  "", // none

	// Debugging fields
	"LogLevel": "INFO",

	// Bind options
	"BindAddress":   "", // none
	"BindInterface": "", // none
}

// GetSSHFieldDefault returns the default value for a given SSH field
// Returns empty string if no default is defined
func GetSSHFieldDefault(fieldName string) string {
	if value, exists := SSHFieldDefaults[fieldName]; exists {
		return value
	}
	return ""
}

// GetSSHFieldDefaultWithFallback returns the default value for a given SSH field
// with a fallback value if no default is defined
func GetSSHFieldDefaultWithFallback(fieldName, fallback string) string {
	if value, exists := SSHFieldDefaults[fieldName]; exists {
		return value
	}
	return fallback
}

// GetFieldPlaceholder returns an appropriate placeholder for a form field
// It returns either the default value, an example, or an empty string
//
//nolint:gocyclo // This is a simple switch statement for field-specific placeholders
func GetFieldPlaceholder(fieldName string) string {
	defaultValue := GetSSHFieldDefault(fieldName)

	switch fieldName {
	// Required fields
	case "Alias", "Host":
		return "required"

	// Fields that show default value in placeholder
	case "Port":
		return "default: " + defaultValue
	case "User":
		return "default: current username"
	case "ConnectTimeout":
		if defaultValue == "" {
			return "seconds (default: none)"
		}
		return "default: " + defaultValue + " seconds"
	case "ConnectionAttempts":
		return "default: " + defaultValue
	case "ServerAliveInterval":
		if defaultValue == "0" {
			return "seconds (default: 0)"
		}
		return "default: " + defaultValue + " seconds"
	case "ServerAliveCountMax":
		return "default: " + defaultValue
	case "NumberOfPasswordPrompts":
		return "default: " + defaultValue
	case "CanonicalizeMaxDots":
		return "default: " + defaultValue
	case "IPQoS":
		return "default: " + defaultValue
	case "EscapeChar":
		return "default: " + defaultValue
	case "IdentityAgent":
		if defaultValue != "" {
			return "default: " + defaultValue
		}
		return "default: SSH_AUTH_SOCK"
	case "UserKnownHostsFile":
		if defaultValue != "" {
			return "default: " + defaultValue
		}
		return "default: ~/.ssh/known_hosts"

	// Fields that show examples in placeholder
	case "Keys":
		return "e.g., ~/.ssh/id_rsa, ~/.ssh/id_ed25519"
	case "Tags":
		return "comma-separated tags"
	case "Group":
		return "e.g., production, database"
	case "ProxyJump": //nolint:goconst // Field name used in switch case
		return "e.g., bastion.example.com"
	case "ProxyCommand":
		return "e.g., ssh -W %h:%p jump.example.com"
	case "RemoteCommand":
		return "e.g., tmux attach"
	case "LocalForward":
		return "e.g., 8080:localhost:80, 3000:localhost:3000"
	case "RemoteForward":
		return "e.g., 80:localhost:8080"
	case "DynamicForward":
		return "e.g., 1080, 1081"
	case "ControlPath":
		return "e.g., ~/.ssh/master-%r@%h:%p"
	case "ControlPersist":
		return "e.g., 10m, 4h, yes, no"
	case "PreferredAuthentications":
		return "e.g., publickey,password"
	case "PubkeyAcceptedAlgorithms", "HostbasedAcceptedAlgorithms",
		"HostKeyAlgorithms", "Ciphers", "MACs", "KexAlgorithms":
		return "algorithms (+/-/^ prefix supported)"
	case "BindAddress":
		return "IP, hostname, * (all), or localhost"
	case "CanonicalDomains":
		return "e.g., example.com, internal.net"
	case "CanonicalizePermittedCNAMEs":
		return "e.g., *.example.com:example.net"
	case "LocalCommand":
		return "e.g., echo 'Connected to %h'"
	case "SendEnv":
		return "e.g., LANG, LC_*, TERM"
	case "SetEnv":
		return "e.g., FOO=bar, DEBUG=1"

	// Fields with no placeholder
	default:
		return ""
	}
}
