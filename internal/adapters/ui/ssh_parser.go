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

// ParseSSHCommand parses an SSH command string and returns ServerFormData
// Supported formats:
//   - ssh user@host
//   - ssh host
//   - ssh user@host -p port
//   - ssh user@host -i keyfile
//   - ssh -p port user@host
//   - ssh -i keyfile user@host
func ParseSSHCommand(cmd string) (ServerFormData, error) {
	data := ServerFormData{
		User: "root",
		Port: "22",
		Key:  "~/.ssh/id_ed25519",
	}

	// Trim whitespace and split into tokens
	cmd = strings.TrimSpace(cmd)
	tokens := strings.Fields(cmd)

	if len(tokens) == 0 {
		return data, fmt.Errorf("empty command")
	}

	// Check if it starts with ssh (optional)
	i := 0
	if tokens[0] == "ssh" {
		i = 1
	}

	if i >= len(tokens) {
		return data, fmt.Errorf("no host specified")
	}

	// Parse arguments and host
	var hostPart string
	for i < len(tokens) {
		token := tokens[i]

		// Handle arguments
		if strings.HasPrefix(token, "-") {
			switch token {
			case "-p":
				// Handle port
				if i+1 < len(tokens) {
					i++
					port, err := strconv.Atoi(tokens[i])
					if err != nil || port < 1 || port > 65535 {
						return data, fmt.Errorf("invalid port: %s", tokens[i])
					}
					data.Port = tokens[i]
				} else {
					return data, fmt.Errorf("missing port value after -p")
				}
			case "-i":
				// Handle key file
				if i+1 < len(tokens) {
					i++
					data.Key = tokens[i]
				} else {
					return data, fmt.Errorf("missing key file after -i")
				}
			default:
				// Ignore other arguments
			}
		} else if hostPart == "" {
			// This should be host or user@host
			hostPart = token
		}
		i++
	}

	if hostPart == "" {
		return data, fmt.Errorf("no host specified")
	}

	// Parse user@host
	if strings.Contains(hostPart, "@") {
		parts := strings.SplitN(hostPart, "@", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			data.User = parts[0]
			data.Host = parts[1]
		} else {
			return data, fmt.Errorf("invalid user@host format: %s", hostPart)
		}
	} else {
		data.Host = hostPart
	}

	// Generate default alias
	if data.Host != "" {
		// If it's an IP address, use it directly
		alias := data.Host
		// Check if it's an IP address
		if !strings.Contains(alias, ":") && strings.Count(alias, ".") == 3 {
			// Might be IPv4, check if all parts are numbers
			parts := strings.Split(alias, ".")
			isIP := true
			for _, part := range parts {
				if _, err := strconv.Atoi(part); err != nil {
					isIP = false
					break
				}
			}
			if !isIP {
				// Not an IP, remove domain suffix to create short alias
				if idx := strings.Index(alias, "."); idx > 0 {
					alias = alias[:idx]
				}
			}
		} else if !strings.Contains(alias, ":") {
			// Not an IP, remove domain suffix to create short alias
			if idx := strings.Index(alias, "."); idx > 0 {
				alias = alias[:idx]
			}
		}
		data.Alias = alias
	}

	return data, nil
}

// BuildServerFromSSHCommand builds a Server object from an SSH command
func BuildServerFromSSHCommand(cmd string) (domain.Server, error) {
	data, err := ParseSSHCommand(cmd)
	if err != nil {
		return domain.Server{}, err
	}

	// Validate form data
	if errMsg := validateServerForm(data); errMsg != "" {
		return domain.Server{}, fmt.Errorf("%s", errMsg)
	}

	// Convert port
	port := 22
	if data.Port != "" {
		if n, err := strconv.Atoi(data.Port); err == nil && n > 0 {
			port = n
		}
	}

	// Process tags
	var tags []string
	if data.Tags != "" {
		for _, t := range strings.Split(data.Tags, ",") {
			if s := strings.TrimSpace(t); s != "" {
				tags = append(tags, s)
			}
		}
	}

	return domain.Server{
		Alias: data.Alias,
		Host:  data.Host,
		User:  data.User,
		Port:  port,
		Key:   data.Key,
		Tags:  tags,
	}, nil
}
