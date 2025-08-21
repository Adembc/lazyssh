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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

const (
	ManagedByComment = "# Managed by lazyssh"
	DefaultPort      = 22
)

type ServerMetadata struct {
	Tags     []string `json:"tags,omitempty"`
	LastSeen string   `json:"last_seen,omitempty"`
	PinnedAt string   `json:"pinned_at,omitempty"`
	SSHCount int      `json:"ssh_count,omitempty"`
}

type serverRepo struct {
	sshConfigFilePath string
	metaDataFilePath  string
	logger            *zap.SugaredLogger
}

func NewServerRepo(logger *zap.SugaredLogger, sshPath, metaDataPath string) *serverRepo {
	return &serverRepo{logger: logger, sshConfigFilePath: sshPath, metaDataFilePath: metaDataPath}
}

func (s serverRepo) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.parseSSHConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	metadata, err := s.loadMetadata()
	if err != nil {
		// Log error but continue without metadata
		metadata = make(map[string]ServerMetadata)
	}

	// Merge metadata with servers
	for i, server := range servers {
		servers[i].LastSeen = time.Time{}

		if meta, exists := metadata[server.Alias]; exists {
			servers[i].Tags = meta.Tags
			servers[i].SSHCount = meta.SSHCount
			if meta.LastSeen != "" {
				if lastSeen, err := time.Parse(time.RFC3339, meta.LastSeen); err == nil {
					servers[i].LastSeen = lastSeen
				}
			}
			if meta.PinnedAt != "" {
				if pinnedAt, err := time.Parse(time.RFC3339, meta.PinnedAt); err == nil {
					servers[i].PinnedAt = pinnedAt
				}
			}
		}
	}

	// Filter by query if provided
	if query != "" {
		filtered := make([]domain.Server, 0)
		queryLower := strings.ToLower(query)
		for _, server := range servers {
			if strings.Contains(strings.ToLower(server.Alias), queryLower) ||
				strings.Contains(strings.ToLower(server.Host), queryLower) ||
				strings.Contains(strings.ToLower(server.User), queryLower) {
				filtered = append(filtered, server)
				continue
			}
			for _, tag := range server.Tags {
				if strings.Contains(strings.ToLower(tag), queryLower) {
					filtered = append(filtered, server)
					break
				}
			}
		}
		return filtered, nil
	}

	return servers, nil
}

func (s serverRepo) UpdateServer(server domain.Server, newServer domain.Server) error {
	servers, err := s.parseSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Find and update the server
	found := false
	for i, srv := range servers {
		if srv.Alias == server.Alias {
			servers[i] = newServer
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}

	// Write back to SSH config
	if err := s.writeSSHConfig(servers); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	// Update metadata
	return s.updateMetadata(newServer)
}

func (s serverRepo) AddServer(server domain.Server) error {
	servers, err := s.parseSSHConfig()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Check if server already exists
	for _, srv := range servers {
		if srv.Alias == server.Alias {
			return fmt.Errorf("server with alias '%s' already exists", server.Alias)
		}
	}

	// Add new server
	servers = append(servers, server)

	// Write to SSH config
	if err := s.writeSSHConfig(servers); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	// Update metadata
	return s.updateMetadata(server)
}

func (s serverRepo) DeleteServer(server domain.Server) error {
	servers, err := s.parseSSHConfig()
	if err != nil {
		return fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Find and remove the server
	found := false
	newServers := make([]domain.Server, 0, len(servers))
	for _, srv := range servers {
		if srv.Alias != server.Alias {
			newServers = append(newServers, srv)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}

	// Write back to SSH config
	if err := s.writeSSHConfig(newServers); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	// Remove from metadata
	return s.removeFromMetadata(server.Alias)
}

func (s serverRepo) parseSSHConfig() ([]domain.Server, error) {
	file, err := os.Open(s.sshConfigFilePath)
	if err != nil {

		if os.IsNotExist(err) {
			return []domain.Server{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var servers []domain.Server
	var currentServer *domain.Server
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split key-value pairs
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		key := strings.ToLower(parts[0])
		value := strings.Join(parts[1:], " ")

		switch key {
		case "host":
			// Save previous server if exists
			if currentServer != nil {
				servers = append(servers, *currentServer)
			}
			// Start new server
			currentServer = &domain.Server{
				Alias: value,
				Port:  DefaultPort, // Default SSH port
			}
		case "hostname":
			if currentServer != nil {
				currentServer.Host = value
			}
		case "user":
			if currentServer != nil {
				currentServer.User = value
			}
		case "port":
			if currentServer != nil {
				if port, err := strconv.Atoi(value); err == nil {
					currentServer.Port = port
				}
			}
		case "identityfile":
			if currentServer != nil {
				// Expand ~ to home directory
				if strings.HasPrefix(value, "~/") {
					if home, err := os.UserHomeDir(); err == nil {
						value = filepath.Join(home, value[2:])
					}
				}
				currentServer.Key = value
			}
		}
	}

	if currentServer != nil {
		servers = append(servers, *currentServer)
	}

	return servers, scanner.Err()
}

func (s serverRepo) writeSSHConfig(servers []domain.Server) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.sshConfigFilePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	file, err := os.Create(s.sshConfigFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for i, server := range servers {
		if i > 0 {
			writer.WriteString("\n")
		}

		fmt.Fprintf(writer, "%s\n", ManagedByComment)
		fmt.Fprintf(writer, "Host %s\n", server.Alias)

		if server.Host != "" {
			fmt.Fprintf(writer, "    HostName %s\n", server.Host)
		}

		if server.User != "" {
			fmt.Fprintf(writer, "    User %s\n", server.User)
		}

		if server.Port != 0 && server.Port != DefaultPort {
			fmt.Fprintf(writer, "    Port %d\n", server.Port)
		}

		if server.Key != "" {
			fmt.Fprintf(writer, "    IdentityFile %s\n", server.Key)
		}
	}

	return nil
}

func (s serverRepo) loadMetadata() (map[string]ServerMetadata, error) {
	metadata := make(map[string]ServerMetadata)

	if _, err := os.Stat(s.metaDataFilePath); os.IsNotExist(err) {
		return metadata, nil
	}

	data, err := os.ReadFile(s.metaDataFilePath)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return metadata, nil
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return metadata, nil
}

func (s serverRepo) updateMetadata(server domain.Server) error {
	metadata, err := s.loadMetadata()
	if err != nil {
		metadata = make(map[string]ServerMetadata)
	}

	serverMeta := ServerMetadata{
		Tags:     server.Tags,
		LastSeen: server.LastSeen.Format(time.RFC3339),
	}

	// Only set PinnedAt if it is not zero
	if !server.PinnedAt.IsZero() {
		serverMeta.PinnedAt = server.PinnedAt.Format(time.RFC3339)
	}

	metadata[server.Alias] = serverMeta

	return s.saveMetadata(metadata)
}

func (s serverRepo) removeFromMetadata(alias string) error {
	metadata, err := s.loadMetadata()
	if err != nil {
		return nil // If we can't load metadata, there's nothing to remove
	}

	delete(metadata, alias)
	return s.saveMetadata(metadata)
}

func (s serverRepo) saveMetadata(metadata map[string]ServerMetadata) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.metaDataFilePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.metaDataFilePath, data, 0o600)
}

// SetPinned updates the pinned state for a server alias by writing pinned_at to metadata.
func (s serverRepo) SetPinned(alias string, pinned bool) error {
	metadata, err := s.loadMetadata()
	if err != nil {
		metadata = make(map[string]ServerMetadata)
	}
	m := metadata[alias]
	if pinned {
		m.PinnedAt = time.Now().Format(time.RFC3339)
	} else {
		m.PinnedAt = ""
	}
	metadata[alias] = m
	return s.saveMetadata(metadata)
}

// RecordSSH updates last_seen and increments ssh_count for an alias after successful SSH.
func (s serverRepo) RecordSSH(alias string) error {
	metadata, err := s.loadMetadata()
	if err != nil {
		metadata = make(map[string]ServerMetadata)
	}
	m := metadata[alias]
	m.LastSeen = time.Now().Format(time.RFC3339)
	m.SSHCount++
	metadata[alias] = m
	return s.saveMetadata(metadata)
}
