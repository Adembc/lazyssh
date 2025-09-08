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
	sshConfigInclude    = "include"
)

type SSHConfigParser struct{}

// Parse parses the main SSH config file and any files included via 'Include' directives.
func (p *SSHConfigParser) Parse(mainConfigPath string) ([]domain.Server, map[string][]string, error) {
	// 1. Find all config files to parse
	filesToParse, err := p.getConfigFiles(mainConfigPath)
	if err != nil {
		return nil, nil, err
	}

	// 2. Parse all files
	var allServers []domain.Server
	includesByFile := make(map[string][]string)
	for _, file := range filesToParse {
		servers, includes, err := p.parseFile(file.Path, file.Group)
		if err != nil {
			// Log or handle error for a single file, maybe continue
			continue
		}
		allServers = append(allServers, servers...)
		if len(includes) > 0 {
			includesByFile[file.Path] = includes
		}
	}

	return allServers, includesByFile, nil
}

type configFile struct {
	Path  string
	Group string
}

func (p *SSHConfigParser) getConfigFiles(mainConfigPath string) ([]configFile, error) {
	var files []configFile
	files = append(files, configFile{Path: mainConfigPath, Group: ""}) // Main config has no group

	// #nosec G304 -- mainConfigPath is trusted
	file, err := os.Open(mainConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil // Main config doesn't exist, nothing to do
		}
		return nil, err
	}
	defer func() { _ = file.Close() }()

	configDir := filepath.Dir(mainConfigPath)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if p.shouldSkipLine(line) {
			continue
		}

		key, value := p.parseKeyValue(line)
		if key == sshConfigInclude {
			includePath := filepath.Join(configDir, value)
			matches, err := filepath.Glob(includePath)
			if err != nil {
				// Log or handle glob error, maybe continue
				continue
			}
			for _, match := range matches {
				group := filepath.Base(match)
				files = append(files, configFile{Path: match, Group: group})
			}
		}
	}

	return files, scanner.Err()
}

func (p *SSHConfigParser) parseFile(path string, group string) (servers []domain.Server, includes []string, err error) {
	// #nosec G304 -- path is generated from globbing trusted config files
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Server{}, []string{}, nil
		}
		return nil, nil, err
	}
	defer func() { _ = file.Close() }()

	servers = make([]domain.Server, 0)
	includes = make([]string, 0)
	var currentServer *domain.Server

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		originalLine := scanner.Text() // Keep original line for includes
		line := strings.TrimSpace(originalLine)

		if p.shouldSkipLine(line) {
			continue
		}

		key, value := p.parseKeyValue(line)
		if key == "" {
			continue
		}

		if key == sshConfigInclude {
			includes = append(includes, originalLine) // Store the original 'Include ...' line
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
				Group: group,
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

	return servers, includes, scanner.Err()
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
