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
)

// GenerateUniqueAlias generates a unique alias by appending a suffix if necessary
func GenerateUniqueAlias(baseAlias string, existingAliases []string) string {
	// Build a map for faster lookup
	aliasMap := make(map[string]bool, len(existingAliases))
	for _, alias := range existingAliases {
		aliasMap[alias] = true
	}

	// If the base alias is unique, return it
	if !aliasMap[baseAlias] {
		return baseAlias
	}

	// Extract the base name and current suffix (if any)
	// e.g., "server_1" -> base: "server", suffix: 1
	// e.g., "server" -> base: "server", suffix: 0
	baseName := baseAlias

	// Check if the alias already has a numeric suffix
	if lastUnderscore := strings.LastIndex(baseAlias, "_"); lastUnderscore != -1 {
		possibleSuffix := baseAlias[lastUnderscore+1:]
		if _, err := strconv.Atoi(possibleSuffix); err == nil {
			// It has a valid numeric suffix
			baseName = baseAlias[:lastUnderscore]
		}
	}

	// Find the highest suffix number for this base name
	maxSuffix := 0
	basePattern := baseName + "_"

	// Check all existing aliases to find the max suffix
	for _, alias := range existingAliases {
		if alias == baseName {
			// The base name without suffix exists, so we need at least _1
			if maxSuffix < 0 {
				maxSuffix = 0
			}
		} else if strings.HasPrefix(alias, basePattern) {
			suffixStr := strings.TrimPrefix(alias, basePattern)
			if suffix, err := strconv.Atoi(suffixStr); err == nil && suffix > maxSuffix {
				maxSuffix = suffix
			}
		}
	}

	// Return the base name with the next available suffix
	return fmt.Sprintf("%s_%d", baseName, maxSuffix+1)
}

// GenerateSmartAlias generates a smart alias from host, user, and port
// It simplifies domain names and handles IP addresses intelligently
func GenerateSmartAlias(host, user string, port int) string {
	alias := host

	// Simplify common domain patterns
	// Remove www. prefix
	alias = strings.TrimPrefix(alias, "www.")

	// For FQDN, extract the meaningful part
	// e.g., server.example.com -> server or example
	if strings.Contains(alias, ".") && !IsIPAddress(alias) {
		parts := strings.Split(alias, ".")
		// If it has subdomain, use the subdomain
		if len(parts) > 2 {
			// e.g., api.github.com -> api.github
			// or dev.server.example.com -> dev.server
			if parts[0] != "www" {
				alias = strings.Join(parts[:2], ".")
			} else {
				alias = parts[1] // Skip www
			}
		} else if len(parts) == 2 {
			// Simple domain like example.com -> example
			alias = parts[0]
		}
	}

	// Optionally prepend user if it's not a common one
	if user != "" && !isCommonUser(user) {
		alias = fmt.Sprintf("%s@%s", user, alias)
	}

	// Append port if non-standard
	if port != 0 && port != 22 {
		alias = fmt.Sprintf("%s:%d", alias, port)
	}

	return alias
}

// isCommonUser checks if a username is a common default username
func isCommonUser(user string) bool {
	// Common usernames that don't need to be included in alias
	// These are default users for various cloud providers and Linux distributions
	commonUsers := map[string]bool{
		// Common administrative users
		"root":          true,
		"admin":         true,
		"administrator": true,

		// Linux distribution default users
		"ubuntu": true, // Ubuntu
		"debian": true, // Debian
		"centos": true, // CentOS
		"fedora": true, // Fedora
		"alpine": true, // Alpine Linux
		"arch":   true, // Arch Linux

		// Cloud provider default users
		"ec2-user":   true, // AWS Amazon Linux
		"azureuser":  true, // Azure
		"opc":        true, // Oracle Cloud
		"cloud-user": true, // Various cloud images
		"cloud_user": true, // Alternative format

		// Container/orchestration platforms
		"core":    true, // CoreOS
		"rancher": true, // RancherOS
		"docker":  true, // Docker

		// Other common defaults
		"user":    true, // Generic
		"guest":   true, // Guest account
		"vagrant": true, // Vagrant boxes
	}

	return commonUsers[strings.ToLower(user)]
}
