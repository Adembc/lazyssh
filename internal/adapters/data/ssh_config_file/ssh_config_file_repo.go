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

package ssh_config_file

import (
	"fmt"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/kevinburke/ssh_config"
	"go.uber.org/zap"
)

// Repository implements ServerRepository interface for SSH config file operations.
type Repository struct {
	configPath      string
	fileSystem      FileSystem
	metadataManager *metadataManager
	logger          *zap.SugaredLogger
}

// NewRepository creates a new SSH config repository.
func NewRepository(logger *zap.SugaredLogger, configPath, metaDataPath string) ports.ServerRepository {
	return &Repository{
		logger:          logger,
		configPath:      configPath,
		fileSystem:      DefaultFileSystem{},
		metadataManager: newMetadataManager(metaDataPath, logger),
	}
}

// NewRepositoryWithFS creates a new SSH config repository with a custom filesystem.
func NewRepositoryWithFS(logger *zap.SugaredLogger, configPath string, metaDataPath string, fs FileSystem) ports.ServerRepository {
	return &Repository{
		logger:          logger,
		configPath:      configPath,
		fileSystem:      fs,
		metadataManager: newMetadataManager(metaDataPath, logger),
	}
}

// ListServers returns all servers matching the query pattern.
// Empty query returns all servers.
func (r *Repository) ListServers(query string) ([]domain.Server, error) {
	cfg, err := r.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	servers := r.toDomainServer(cfg)
	metadata, err := r.metadataManager.loadAll()
	if err != nil {
		r.logger.Warnf("Failed to load metadata: %v", err)
		metadata = make(map[string]ServerMetadata)
	}
	servers = r.mergeMetadata(servers, metadata)
	if query == "" {
		return servers, nil
	}

	return r.filterServers(servers, query), nil
}

// AddServer adds a new server to the SSH config.
func (r *Repository) AddServer(server domain.Server) error {
	cfg, err := r.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if r.serverExists(cfg, server.Alias) {
		return fmt.Errorf("server with alias '%s' already exists", server.Alias)
	}

	host := r.createHostFromServer(server)
	cfg.Hosts = append(cfg.Hosts, host)

	if err := r.saveConfig(cfg); err != nil {
		r.logger.Warnf("Failed to save config while adding new server: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}
	return r.metadataManager.updateServer(server, server.Alias)
}

// UpdateServer updates an existing server in the SSH config.
func (r *Repository) UpdateServer(server domain.Server, newServer domain.Server) error {
	cfg, err := r.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	host := r.findHostByAlias(cfg, server.Alias)
	if host == nil {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}

	if server.Alias != newServer.Alias {
		if r.serverExists(cfg, newServer.Alias) {
			return fmt.Errorf("server with alias '%s' already exists", newServer.Alias)
		}

		newPatterns := make([]*ssh_config.Pattern, 0, len(host.Patterns))
		for _, pattern := range host.Patterns {
			if pattern.Str == server.Alias {
				newPatterns = append(newPatterns, &ssh_config.Pattern{Str: newServer.Alias})
			} else {
				newPatterns = append(newPatterns, pattern)
			}
		}

		host.Patterns = newPatterns

	}

	r.updateHostNodes(host, newServer)

	if err := r.saveConfig(cfg); err != nil {
		r.logger.Warnf("Failed to save config while updating server: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}
	// Update metadata; pass old alias to allow inline migration
	return r.metadataManager.updateServer(newServer, server.Alias)
}

// DeleteServer removes a server from the SSH config.
func (r *Repository) DeleteServer(server domain.Server) error {
	cfg, err := r.loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	initialCount := len(cfg.Hosts)
	cfg.Hosts = r.removeHostByAlias(cfg.Hosts, server.Alias)

	if len(cfg.Hosts) == initialCount {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}

	if err := r.saveConfig(cfg); err != nil {
		r.logger.Warnf("Failed to save config while deleting server: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}
	return r.metadataManager.deleteServer(server.Alias)
}

// SetPinned sets or unsets the pinned status of a server.
func (r *Repository) SetPinned(alias string, pinned bool) error {
	return r.metadataManager.setPinned(alias, pinned)
}

// RecordSSH increments the SSH access count and updates the last seen timestamp for a server.
func (r *Repository) RecordSSH(alias string) error {
	return r.metadataManager.recordSSH(alias)
}
