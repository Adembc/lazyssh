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
	"strings"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

type serverRepo struct {
	sshConfigManager *sshConfigManager
	awsConfigManager *AWSConfigManager
	metadataManager  *metadataManager
	configManager    *configManager
	configDirPath    string
	logger           *zap.SugaredLogger
}

func NewServerRepo(logger *zap.SugaredLogger, sshPath, metaDataPath, configPath string) *serverRepo {
	return NewServerRepoWithConfigDir(logger, sshPath, metaDataPath, configPath, "")
}

func NewServerRepoWithConfigDir(logger *zap.SugaredLogger, sshPath, metaDataPath, configPath, configDirPath string) *serverRepo {
	configManager := newConfigManager(configPath)
	
	// Load configuration to get AWS paths
	config, err := configManager.load()
	if err != nil {
		logger.Warnf("Failed to load configuration, using defaults: %v", err)
		config = domain.DefaultConfig(configDirPath)
	}
	
	// The AWS servers path is already set correctly by the config manager
	// which now defaults to configDirPath/aws-servers.yaml
	awsServersPath := config.AWSServersPath
	
	// Expand home directory in paths if needed (for legacy paths)
	if strings.Contains(awsServersPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Warnf("Failed to get home directory: %v", err)
			home = "~" // fallback
		}
		awsServersPath = strings.ReplaceAll(awsServersPath, "~", home)
	}
	
	return &serverRepo{
		sshConfigManager: newSSHConfigManager(sshPath),
		awsConfigManager: newAWSConfigManager(awsServersPath),
		metadataManager:  newMetadataManager(metaDataPath),
		configManager:    configManager,
		configDirPath:    configDirPath,
		logger:           logger,
	}
}

func (s *serverRepo) ListServers(query string) ([]domain.Server, error) {
	// Load configuration
	config, err := s.configManager.load()
	if err != nil {
		s.logger.Warnf("Failed to load configuration, using defaults: %v", err)
		config = domain.DefaultConfig(s.configDirPath)
	}

	// Parse SSH servers
	sshServers, err := s.sshConfigManager.parseServers()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH config: %w", err)
	}

	// Set SSH server properties
	for i := range sshServers {
		sshServers[i].ConnectionType = domain.ConnectionTypeSSH
		sshServers[i].Source = "ssh_config"
	}

	// Start with SSH servers
	allServers := sshServers

	// Conditionally parse AWS servers based on configuration
	if config.EnableAWSSource {
		awsServers, err := s.awsConfigManager.parseServers()
		if err != nil {
			s.logger.Warnf("Failed to parse AWS configurations: %v", err)
		} else {
			// Combine SSH and AWS servers
			allServers = append(allServers, awsServers...)
		}
	}

	// Load and merge metadata
	metadata, err := s.metadataManager.loadAll()
	if err != nil {
		s.logger.Warnf("Failed to load metadata: %v", err)
		metadata = make(map[string]ServerMetadata)
	}

	allServers = s.mergeMetadata(allServers, metadata)

	// Apply query filter if provided
	if query != "" {
		allServers = s.filterServers(allServers, query)
	}

	return allServers, nil
}

func (s *serverRepo) UpdateServer(server domain.Server, newServer domain.Server) error {
	// Only allow updates to SSH servers, not AWS servers
	if server.ConnectionType == domain.ConnectionTypeAWS {
		return fmt.Errorf("cannot update AWS servers - they are managed through the AWS YAML configuration file")
	}

	if err := s.sshConfigManager.updateServer(server.Alias, newServer); err != nil {
		return fmt.Errorf("failed to update SSH config: %w", err)
	}

	return s.metadataManager.updateServer(newServer)
}

func (s *serverRepo) AddServer(server domain.Server) error {
	// Force new servers to be SSH type
	server.ConnectionType = domain.ConnectionTypeSSH
	server.Source = "ssh_config"

	if err := s.sshConfigManager.addServer(server); err != nil {
		return fmt.Errorf("failed to add to SSH config: %w", err)
	}

	return s.metadataManager.updateServer(server)
}

func (s *serverRepo) DeleteServer(server domain.Server) error {
	// Only allow deletion of SSH servers, not AWS servers
	if server.ConnectionType == domain.ConnectionTypeAWS {
		return fmt.Errorf("cannot delete AWS servers - they are managed through the AWS YAML configuration file")
	}

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

func (s *serverRepo) SetAWSSourceEnabled(enabled bool) error {
	config, err := s.configManager.load()
	if err != nil {
		s.logger.Warnf("Failed to load configuration, using defaults: %v", err)
		config = domain.DefaultConfig(s.configDirPath)
	}

	config.EnableAWSSource = enabled
	return s.configManager.save(config)
}

func (s *serverRepo) SetAWSServersPath(path string) error {
	config, err := s.configManager.load()
	if err != nil {
		s.logger.Warnf("Failed to load configuration, using defaults: %v", err)
		config = domain.DefaultConfig(s.configDirPath)
	}

	config.AWSServersPath = path
	if err := s.configManager.save(config); err != nil {
		return err
	}

	// Update the AWS config manager with the new path
	awsServersPath := path
	
	// Expand home directory in path if needed
	if strings.Contains(awsServersPath, "~") {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			s.logger.Warnf("Failed to get home directory: %v", homeErr)
			home = "~" // fallback
		}
		awsServersPath = strings.ReplaceAll(awsServersPath, "~", home)
	}
	
	s.awsConfigManager = newAWSConfigManager(awsServersPath)
	
	return nil
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
		strings.Contains(strings.ToLower(server.User), queryLower) {
		return true
	}

	for _, tag := range server.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return true
		}
	}

	return false
}
