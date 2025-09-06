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

package file

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

const (
	sshConfigAliasField = "host"
	sshConfigIPField    = "hostname"
	sshConfigUserField  = "user"
	sshConfigPortField  = "port"
	sshConfigKeyField   = "identityfile"
)

type SSHConfigParser struct{}

func (p *SSHConfigParser) Parse(reader io.Reader) ([]domain.Server, error) {
	servers := make([]domain.Server, 0)
	var currentServer *domain.Server

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Handle tags and info comments
		commentKey, commentValue := p.parseComments(line)
		if commentKey != "" && currentServer != nil {
			switch commentKey {
			case "tags":
				for _, tag := range strings.Split(commentValue, ",") {
					currentServer.Tags = append(currentServer.Tags, strings.TrimSpace(tag))
				}
			case "info":
				currentServer.Info = commentValue
			}
			continue
		}

		key, value := p.parseKeyValue(line)
		if key == "" {
			continue
		}

		switch key {
		case sshConfigAliasField:
			if currentServer != nil {
				servers = append(servers, *currentServer)
			}
			currentServer = &domain.Server{
				Alias: value,
				Port:  DefaultPort,
			}
		case sshConfigIPField:
			if currentServer != nil {
				currentServer.Host = value
			}
		case sshConfigUserField:
			if currentServer != nil {
				currentServer.User = value
			}
		case sshConfigPortField:
			if currentServer != nil {
				currentServer.Port = p.parsePort(value)
			}
		case sshConfigKeyField:
			if currentServer != nil {
				currentServer.Key = p.expandPath(value)
			}
		}
	}

	if currentServer != nil {
		servers = append(servers, *currentServer)
	}

	return servers, scanner.Err()
}

func (p *SSHConfigParser) parseComments(line string) (string, string) {
	tagPrefixes := []string{"#tag:", "#tags:"}
	for _, prefix := range tagPrefixes {
		if strings.HasPrefix(line, prefix) {
			return "tags", strings.TrimSpace(line[len(prefix):])
		}
	}
	infoPrefix := "#info:"
	if strings.HasPrefix(line, infoPrefix) {
		return "info", strings.TrimSpace(line[len(infoPrefix):])
	}
	return "", ""
}

func (p *SSHConfigParser) shouldSkipLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func (p *SSHConfigParser) parseKeyValue(line string) (string, string) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", ""
	}

	key := strings.ToLower(parts[0])
	value := strings.Join(parts[1:], " ")
	return key, value
}

func (p *SSHConfigParser) parsePort(value string) int {
	if port, err := strconv.Atoi(value); err == nil {
		return port
	}
	return DefaultPort
}

func (p *SSHConfigParser) expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
