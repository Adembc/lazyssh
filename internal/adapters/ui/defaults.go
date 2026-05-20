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
	"github.com/Adembc/lazyssh/internal/i18n"
)

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
		return i18n.T("placeholder.required")

	// Fields that show default value in placeholder
	case "Port":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "User":
		return i18n.T("placeholder.default") + ": " + i18n.T("placeholder.current_user")
	case "ConnectTimeout":
		if defaultValue == "" {
			return i18n.T("placeholder.seconds") + " (" + i18n.T("placeholder.default") + ": " + i18n.T("placeholder.none") + ")"
		}
		return i18n.T("placeholder.default") + ": " + defaultValue + " " + i18n.T("placeholder.seconds")
	case "ConnectionAttempts":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "ServerAliveInterval":
		if defaultValue == "0" {
			return i18n.T("placeholder.seconds") + " (" + i18n.T("placeholder.default") + ": 0)"
		}
		return i18n.T("placeholder.default") + ": " + defaultValue + " " + i18n.T("placeholder.seconds")
	case "ServerAliveCountMax":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "NumberOfPasswordPrompts":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "CanonicalizeMaxDots":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "IPQoS":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "EscapeChar":
		return i18n.T("placeholder.default") + ": " + defaultValue
	case "IdentityAgent":
		if defaultValue != "" {
			return i18n.T("placeholder.default") + ": " + defaultValue
		}
		return i18n.T("placeholder.default") + ": SSH_AUTH_SOCK"
	case "UserKnownHostsFile":
		if defaultValue != "" {
			return i18n.T("placeholder.default") + ": " + defaultValue
		}
		return i18n.T("placeholder.default") + ": ~/.ssh/known_hosts"

	// Fields that show examples in placeholder
	case "Keys":
		return i18n.T("placeholder_keys")
	case "Tags":
		return i18n.T("placeholder.comma_separated")
	case "Password":
		return i18n.T("field.placeholder.password")
	case "ProxyJump": //nolint:goconst // Field name used in switch case
		return i18n.T("placeholder.proxy_jump")
	case "ProxyCommand":
		return i18n.T("placeholder.proxy_command")
	case "RemoteCommand":
		return i18n.T("placeholder.remote_command")
	case "LocalForward":
		return i18n.T("placeholder.local_forward")
	case "RemoteForward":
		return i18n.T("placeholder.remote_forward")
	case "DynamicForward":
		return i18n.T("placeholder.dynamic_forward")
	case "ControlPath":
		return i18n.T("placeholder_control_path")
	case "ControlPersist":
		return i18n.T("placeholder.control_persist")
	case "PreferredAuthentications":
		return i18n.T("placeholder.preferred_auth")
	case "PubkeyAcceptedAlgorithms", "HostbasedAcceptedAlgorithms",
		"HostKeyAlgorithms", "Ciphers", "MACs", "KexAlgorithms":
		return i18n.T("placeholder_algorithms")
	case "BindAddress":
		return i18n.T("placeholder_bind_address")
	case "CanonicalDomains":
		return i18n.T("placeholder.canonical_domains")
	case "CanonicalizePermittedCNAMEs":
		return i18n.T("placeholder.canonical_cname")
	case "LocalCommand":
		return i18n.T("placeholder.local_command")
	case "SendEnv":
		return i18n.T("placeholder.send_env")
	case "SetEnv":
		return i18n.T("placeholder_set_env")

	// Fields with no placeholder
	default:
		return ""
	}
}
