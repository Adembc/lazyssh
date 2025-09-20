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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

type ServerMetadata struct {
	Tags     []string `json:"tags,omitempty"`
	LastSeen string   `json:"last_seen,omitempty"`
	PinnedAt string   `json:"pinned_at,omitempty"`
	SSHCount int      `json:"ssh_count,omitempty"`
	Hidden   bool     `json:"hidden,omitempty"`
}

type metadataManager struct {
	filePath string
	logger   *zap.SugaredLogger
}

func newMetadataManager(filePath string, logger *zap.SugaredLogger) *metadataManager {
	return &metadataManager{filePath: filePath, logger: logger}
}

func (m *metadataManager) loadAll() (map[string]ServerMetadata, error) {
	metadata := make(map[string]ServerMetadata)

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return metadata, nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("read metadata '%s': %w", m.filePath, err)
	}

	if len(data) == 0 {
		return metadata, nil
	}

	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata JSON '%s': %w", m.filePath, err)
	}

	return metadata, nil
}

func (m *metadataManager) saveAll(metadata map[string]ServerMetadata) error {
	if err := m.ensureDirectory(); err != nil {
		m.logger.Errorw("failed to ensure metadata directory", "path", m.filePath, "error", err)

		return fmt.Errorf("ensure metadata directory for '%s': %w", m.filePath, err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		m.logger.Errorw("failed to marshal metadata", "path", m.filePath, "error", err)
		return fmt.Errorf("marshal metadata for '%s': %w", m.filePath, err)
	}

	if err := os.WriteFile(m.filePath, data, 0o600); err != nil {
		m.logger.Errorw("failed to write metadata file", "path", m.filePath, "error", err)
		return fmt.Errorf("write metadata '%s': %w", m.filePath, err)
	}
	return nil
}

func (m *metadataManager) updateServer(server domain.Server, oldAlias string) error {
	metadata, err := m.loadAll()
	if err != nil {
		m.logger.Errorw("failed to load metadata in updateServer", "path", m.filePath, "alias", server.Alias, "old_alias", oldAlias, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	if oldAlias != server.Alias {
		oldMeta, ok := metadata[oldAlias]
		if ok {
			metadata[server.Alias] = oldMeta
		}
		delete(metadata, oldAlias)
	}

	existing := metadata[server.Alias]
	merged := existing

	merged.Tags = server.Tags

	if !server.LastSeen.IsZero() {
		merged.LastSeen = server.LastSeen.Format(time.RFC3339)
	}

	if !server.PinnedAt.IsZero() {
		merged.PinnedAt = server.PinnedAt.Format(time.RFC3339)
	}

	if server.SSHCount > 0 {
		merged.SSHCount = server.SSHCount
	}

	metadata[server.Alias] = merged
	return m.saveAll(metadata)
}

func (m *metadataManager) deleteServer(alias string) error {
	metadata, err := m.loadAll()
	if err != nil {
		m.logger.Errorw("failed to load metadata in deleteServer", "path", m.filePath, "alias", alias, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	delete(metadata, alias)
	return m.saveAll(metadata)
}

func (m *metadataManager) setPinned(alias string, pinned bool) error {
	metadata, err := m.loadAll()
	if err != nil {
		m.logger.Errorw("failed to load metadata in setPinned", "path", m.filePath, "alias", alias, "pinned", pinned, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	meta := metadata[alias]
	if pinned {
		meta.PinnedAt = time.Now().Format(time.RFC3339)
	} else {
		meta.PinnedAt = ""
	}

	metadata[alias] = meta
	return m.saveAll(metadata)
}

func (m *metadataManager) setHidden(alias string, hidden bool) error {
	metadata, err := m.loadAll()
	if err != nil {
		m.logger.Errorw("failed to load metadata in setHidden", "path", m.filePath, "alias", alias, "hidden", hidden, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	meta := metadata[alias]
	meta.Hidden = hidden
	metadata[alias] = meta

	return m.saveAll(metadata)
}

func (m *metadataManager) recordSSH(alias string) error {
	metadata, err := m.loadAll()
	if err != nil {
		m.logger.Errorw("failed to load metadata in recordSSH", "path", m.filePath, "alias", alias, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	meta := metadata[alias]
	meta.LastSeen = time.Now().Format(time.RFC3339)
	meta.SSHCount++

	metadata[alias] = meta
	return m.saveAll(metadata)
}

func (m *metadataManager) ensureDirectory() error {
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("mkdir '%s': %w", dir, err)
	}
	return nil
}
