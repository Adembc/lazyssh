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

	"github.com/kjm0001/lazyssh/internal/core/domain"
)

const (
	ManagedByComment = "# Managed by lazyssh"
	DefaultPort      = 22
)

type sshConfigManager struct {
	filePath string
}

func newSSHConfigManager(filePath string) *sshConfigManager {
	return &sshConfigManager{filePath: filePath}
}

func (m *sshConfigManager) parseServers() ([]domain.Server, error) {
	file, err := os.Open(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.Server{}, nil
		}
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	parser := &SSHConfigParser{}
	return parser.Parse(file)
}

func (m *sshConfigManager) writeServers(servers []domain.Server) error {
	if err := m.ensureDirectory(); err != nil {
		return err
	}

	if err := m.backupCurrentConfig(); err != nil {
		return err
	}

	dir := filepath.Dir(m.filePath)
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
	if err := writer.Write(tmp, servers); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil { // close after sync to ensure contents are persisted
		return err
	}

	if err := os.Rename(tmp.Name(), m.filePath); err != nil {
		return err
	}

	return nil
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
	return m.writeServers(servers)
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

	return m.writeServers(servers)
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

	return m.writeServers(newServers)
}

func (m *sshConfigManager) ensureDirectory() error {
	dir := filepath.Dir(m.filePath)
	return os.MkdirAll(dir, 0o700)
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
