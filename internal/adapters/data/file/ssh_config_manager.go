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
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

const (
	ManagedByComment = "# Managed by lazyssh"
	DefaultPort      = 22
)

type sshConfigManager struct {
	filePath       string
	cachedIncludes map[string][]string
}

func newSSHConfigManager(filePath string) *sshConfigManager {
	return &sshConfigManager{
		filePath:       filePath,
		cachedIncludes: make(map[string][]string),
	}
}

func (m *sshConfigManager) parseServers() ([]domain.Server, error) {
	parser := &SSHConfigParser{}
	servers, includes, err := parser.Parse(m.filePath)
	if err != nil {
		return nil, err
	}
	m.cachedIncludes = includes
	return servers, nil
}

func (m *sshConfigManager) writeGroupedServers(servers []domain.Server) error {
	configDir := filepath.Dir(m.filePath)
	groupDir := filepath.Join(configDir, "config.d")

	// Get a list of all group files that currently exist, to detect deletions.
	existingGroupFiles, err := filepath.Glob(filepath.Join(groupDir, "*"))
	if err != nil {
		return fmt.Errorf("failed to list existing group files: %w", err)
	}

	// Group the new state of servers.
	groupedServers := make(map[string][]domain.Server)
	for _, s := range servers {
		groupedServers[s.Group] = append(groupedServers[s.Group], s)
	}

	// Write the current groups to their files.
	for group, srvs := range groupedServers {
		path := m.filePath
		if group != "" {
			path = filepath.Join(groupDir, group)
		}

		// Get the original includes for this specific path from the cache.
		directives := m.cachedIncludes[path]

		if err := m.writeFile(path, srvs, directives); err != nil {
			return fmt.Errorf("failed to write group %s: %w", group, err)
		}
	}

	// Determine which of the old group files should now be deleted.
	for _, oldFile := range existingGroupFiles {
		groupName := filepath.Base(oldFile)
		if _, stillExists := groupedServers[groupName]; !stillExists {
			// This group is no longer in our map, so it's empty. Delete the file.
			if err := os.Remove(oldFile); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove obsolete group file %s: %w", oldFile, err)
			}
		}
	}

	return nil
}

// writeFile writes a list of servers to a specific file path.
func (m *sshConfigManager) writeFile(path string, servers []domain.Server, directives []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	// Note: The backup logic in backupCurrentConfig is tied to m.filePath.
	// For simplicity, we only back up the main config file.
	// A more robust solution might back up each file.
	if path == m.filePath {
		if err := m.backupCurrentConfig(); err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lazyssh-tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		_ = tmp.Close()
		return err
	}

	writer := &SSHConfigWriter{}
	if err := writer.Write(tmp, servers, directives); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), path)
}

func (m *sshConfigManager) addServer(server domain.Server) error {
	servers, err := m.parseServers()
	if err != nil {
		return err
	}

	// Check for duplicates
	for _, srv := range servers {
		if srv.Alias == server.Alias {
			return fmt.Errorf("server with alias '%s' already exists", server.Alias)
		}
	}

	servers = append(servers, server)
	return m.writeGroupedServers(servers)
}

func (m *sshConfigManager) updateServer(alias string, newServer domain.Server) error {
	servers, err := m.parseServers()
	if err != nil {
		return err
	}

	found := false
	for i, srv := range servers {
		if srv.Alias == alias {
			servers[i] = newServer
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("server with alias '%s' not found", alias)
	}

	return m.writeGroupedServers(servers)
}

func (m *sshConfigManager) deleteServer(alias string) error {
	servers, err := m.parseServers()
	if err != nil {
		return err
	}

	newServers := make([]domain.Server, 0, len(servers))
	found := false

	for _, srv := range servers {
		if srv.Alias != alias {
			newServers = append(newServers, srv)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("server with alias '%s' not found", alias)
	}

	return m.writeGroupedServers(newServers)
}

// backupCurrentConfig creates ~/.lazyssh/backups/config.backup with 0600 perms,
// overwriting it each time, but only if the source config exists.
func (m *sshConfigManager) backupCurrentConfig() error {
	// If source config does not exist, skip backup
	if _, err := os.Stat(m.filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	backupDir := filepath.Join(home, ".lazyssh", "backups")
	// Ensure directory with 0700
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, "config.backup")
	// Copy file contents
	src, err := os.Open(m.filePath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	// #nosec G304 -- backupPath is generated internally and trusted
	dst, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}

	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	if err := dst.Sync(); err != nil {
		return err
	}
	return nil
}
