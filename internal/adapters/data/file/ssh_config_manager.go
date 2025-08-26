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
	"os"
	"path/filepath"

	"github.com/Adembc/lazyssh/internal/core/domain"
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

	file, err := os.Create(m.filePath)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	writer := &SSHConfigWriter{}
	return writer.Write(file, servers)
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
