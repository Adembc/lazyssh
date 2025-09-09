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
	"strings"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

type serverRepo struct {
	sshConfigManager *sshConfigManager
	metadataManager  *metadataManager
	logger           *zap.SugaredLogger
}

func NewServerRepo(logger *zap.SugaredLogger, sshPath, metaDataPath string) *serverRepo {
	return &serverRepo{
		sshConfigManager: newSSHConfigManager(sshPath),
		metadataManager:  newMetadataManager(metaDataPath),
		logger:           logger,
	}
}

func (s *serverRepo) ListServers(query string) ([]domain.Server, error) {
	servers, err := s.sshConfigManager.parseServers()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	metadata, err := s.metadataManager.loadAll()
	if err != nil {
		s.logger.Warnf("Failed to load metadata: %v", err)
		metadata = make(map[string]ServerMetadata)
	}

	servers = s.mergeMetadata(servers, metadata)

	if query != "" {
		servers = s.filterServers(servers, query)
	}

	return servers, nil
}

func (s *serverRepo) UpdateServer(server domain.Server, newServer domain.Server) error {
	if err := s.sshConfigManager.updateServer(server.Alias, newServer); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	return s.metadataManager.updateServer(newServer)
}

func (s *serverRepo) AddServer(server domain.Server) error {
	if err := s.sshConfigManager.addServer(server); err != nil {
		return fmt.Errorf("failed to add to SSH config: %w", err)
	}

	return s.metadataManager.updateServer(server)
}

func (s *serverRepo) DeleteServer(server domain.Server) error {
	if err := s.sshConfigManager.deleteServer(server.Alias); err != nil {
		return fmt.Errorf("failed to delete from SSH config: %w", err)
	}

	return s.metadataManager.deleteServer(server.Alias)
}

func (s *serverRepo) SetPinned(alias string, pinned bool) error {
	return s.metadataManager.setPinned(alias, pinned)
}

func (s *serverRepo) RecordSSH(alias string) error {
	return s.metadataManager.recordSSH(alias)
}

func (s *serverRepo) mergeMetadata(servers []domain.Server, metadata map[string]ServerMetadata) []domain.Server {
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
	return servers
}

func (s *serverRepo) filterServers(servers []domain.Server, query string) []domain.Server {
	queryLower := strings.ToLower(query)
	filtered := make([]domain.Server, 0)

	for _, server := range servers {
		if s.matchesQuery(server, queryLower) {
			filtered = append(filtered, server)
		}
	}

	return filtered
}

func (s *serverRepo) matchesQuery(server domain.Server, queryLower string) bool {
	if strings.Contains(strings.ToLower(server.Alias), queryLower) ||
		strings.Contains(strings.ToLower(server.Host), queryLower) ||
		strings.Contains(strings.ToLower(server.User), queryLower) ||
		strings.Contains(strings.ToLower(server.Info), queryLower){
		return true
	}

	for _, tag := range server.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return true
		}
	}

	return false
}
