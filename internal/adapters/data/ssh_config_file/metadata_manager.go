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

// Settings contains application-level settings stored in the metadata file.
type Settings struct {
	Theme string `json:"theme,omitempty"`
}

// ServerMetadata contains per-server metadata that is not part of SSH config.
type ServerMetadata struct {
	Tags     []string `json:"tags,omitempty"`
	LastSeen string   `json:"last_seen,omitempty"`
	PinnedAt string   `json:"pinned_at,omitempty"`
	SSHCount int      `json:"ssh_count,omitempty"`
}

// MetadataFile is the top-level structure of the metadata JSON file.
// It contains both application settings and per-server metadata.
type MetadataFile struct {
	Settings Settings                  `json:"settings,omitempty"`
	Servers  map[string]ServerMetadata `json:"servers,omitempty"`
}

type metadataManager struct {
	filePath string
	logger   *zap.SugaredLogger
}

func newMetadataManager(filePath string, logger *zap.SugaredLogger) *metadataManager {
	return &metadataManager{filePath: filePath, logger: logger}
}

// loadFile loads the entire metadata file, handling both old and new formats.
// Old format: {"server1": {...}, "server2": {...}}
// New format: {"settings": {...}, "servers": {"server1": {...}, ...}}
func (m *metadataManager) loadFile() (*MetadataFile, error) {
	result := &MetadataFile{
		Servers: make(map[string]ServerMetadata),
	}

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return result, nil
	}

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return nil, fmt.Errorf("read metadata '%s': %w", m.filePath, err)
	}

	if len(data) == 0 {
		return result, nil
	}

	// First, try to parse as the new format
	if err := json.Unmarshal(data, result); err != nil {
		return nil, fmt.Errorf("parse metadata JSON '%s': %w", m.filePath, err)
	}

	// Check if this was the old format (no "servers" key, just server entries at root)
	// In the old format, result.Servers will be nil/empty and the root object contains server data
	if len(result.Servers) == 0 {
		// Try parsing as old format (map of server metadata directly)
		var oldFormat map[string]ServerMetadata
		if err := json.Unmarshal(data, &oldFormat); err == nil {
			// Check if this looks like server metadata (has expected fields)
			// and not a settings object
			isOldFormat := false
			for _, v := range oldFormat {
				// If any entry has tags, last_seen, pinned_at, or ssh_count, it's old format
				if len(v.Tags) > 0 || v.LastSeen != "" || v.PinnedAt != "" || v.SSHCount > 0 {
					isOldFormat = true
					break
				}
			}
			if isOldFormat {
				result.Servers = oldFormat
			}
		}
	}

	if result.Servers == nil {
		result.Servers = make(map[string]ServerMetadata)
	}

	return result, nil
}

// saveFile saves the entire metadata file in the new format.
func (m *metadataManager) saveFile(file *MetadataFile) error {
	if err := m.ensureDirectory(); err != nil {
		m.logger.Errorw("failed to ensure metadata directory", "path", m.filePath, "error", err)
		return fmt.Errorf("ensure metadata directory for '%s': %w", m.filePath, err)
	}

	data, err := json.MarshalIndent(file, "", "  ")
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

// GetSettings returns the application settings from the metadata file.
func (m *metadataManager) GetSettings() (Settings, error) {
	file, err := m.loadFile()
	if err != nil {
		return Settings{}, err
	}
	return file.Settings, nil
}

// SaveSettings saves the application settings to the metadata file.
func (m *metadataManager) SaveSettings(settings Settings) error {
	file, err := m.loadFile()
	if err != nil {
		m.logger.Errorw("failed to load metadata in SaveSettings", "path", m.filePath, "error", err)
		return fmt.Errorf("load metadata: %w", err)
	}

	file.Settings = settings
	return m.saveFile(file)
}

func (m *metadataManager) loadAll() (map[string]ServerMetadata, error) {
	file, err := m.loadFile()
	if err != nil {
		return nil, err
	}
	return file.Servers, nil
}

func (m *metadataManager) saveAll(metadata map[string]ServerMetadata) error {
	file, err := m.loadFile()
	if err != nil {
		// If we can't load, start fresh but preserve any settings
		file = &MetadataFile{}
	}

	file.Servers = metadata
	return m.saveFile(file)
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
