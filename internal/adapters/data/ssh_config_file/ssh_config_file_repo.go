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
	lc, err := r.loadConfig()
	if err != nil {
		return nil, err
	}

	servers := r.toDomainServer(lc)
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

// AddServer adds a new server to the SSH config. If server.SourceFile is set
// and matches a loaded file, the new host is written there; otherwise it
// goes into the main config file.
func (r *Repository) AddServer(server domain.Server) error {
	lc, err := r.loadConfig()
	if err != nil {
		return err
	}

	if r.serverExists(lc, server.Alias) {
		return fmt.Errorf("server with alias '%s' already exists", server.Alias)
	}

	target := lc.findFile(server.SourceFile)
	if target == nil {
		// Default: main file.
		target = &lc.files[0]
	}

	host := r.createHostFromServer(server)
	target.cfg.Hosts = append(target.cfg.Hosts, host)

	if err := r.saveFiles(lc, []string{target.path}); err != nil {
		r.logger.Warnf("Failed to save config while adding new server: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}
	return r.metadataManager.updateServer(server, server.Alias)
}

// UpdateServer updates an existing server in the SSH config. The host is
// mutated in whichever file currently defines it (preferring server.SourceFile
// when the alias is defined in multiple files).
func (r *Repository) UpdateServer(server domain.Server, newServer domain.Server) error {
	lc, err := r.loadConfig()
	if err != nil {
		return err
	}

	matches := r.findHostMatches(lc, server.Alias)
	if len(matches) == 0 {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}
	if len(matches) > 1 && !preferenceResolves(matches, server.SourceFile) {
		return &domain.ErrAmbiguousHost{Alias: server.Alias, Candidates: matchPaths(matches)}
	}

	picked, _ := pickWritableMatch(matches, server.SourceFile)
	host := picked.host

	if server.Alias != newServer.Alias {
		if r.serverExists(lc, newServer.Alias) {
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

	if err := r.saveFiles(lc, []string{picked.path}); err != nil {
		r.logger.Warnf("Failed to save config while updating server: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}
	if err := r.metadataManager.updateServer(newServer, server.Alias); err != nil {
		return err
	}
	return r.metadataManager.setFile(newServer.Alias, picked.path)
}

// DeleteServer removes a server from the SSH config (from whichever file
// currently defines it; preferring server.SourceFile on ambiguity).
func (r *Repository) DeleteServer(server domain.Server) error {
	lc, err := r.loadConfig()
	if err != nil {
		return err
	}

	matches := r.findHostMatches(lc, server.Alias)
	if len(matches) == 0 {
		return fmt.Errorf("server with alias '%s' not found", server.Alias)
	}
	if len(matches) > 1 && !preferenceResolves(matches, server.SourceFile) {
		return &domain.ErrAmbiguousHost{Alias: server.Alias, Candidates: matchPaths(matches)}
	}

	picked, _ := pickWritableMatch(matches, server.SourceFile)
	picked.cfg.Hosts = r.removeHostByAlias(picked.cfg.Hosts, server.Alias)

	if err := r.saveFiles(lc, []string{picked.path}); err != nil {
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
