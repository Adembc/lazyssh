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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

type ServerMetadata struct {
	Tags     []string `json:"tags,omitempty"`
	LastSeen string   `json:"last_seen,omitempty"`
	PinnedAt string   `json:"pinned_at,omitempty"`
	SSHCount int      `json:"ssh_count,omitempty"`
}

type metadataManager struct {
	filePath string
}

func newMetadataManager(filePath string) *metadataManager {
	return &metadataManager{filePath: filePath}
}

func (m *metadataManager) loadAll() (map[string]ServerMetadata, error) {
	metadata := make(map[string]ServerMetadata)

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return metadata, nil
	}

	data, err := os.ReadFile(m.filePath)
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

func (m *metadataManager) saveAll(metadata map[string]ServerMetadata) error {
	if err := m.ensureDirectory(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0o600)
}

func (m *metadataManager) updateServer(server domain.Server) error {
	metadata, err := m.loadAll()
	if err != nil {
		metadata = make(map[string]ServerMetadata)
	}

	existing := metadata[server.Alias]
	merged := existing

	if server.Tags != nil {
		merged.Tags = server.Tags
	}

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
		return nil // Nothing to delete if we can't load
	}

	delete(metadata, alias)
	return m.saveAll(metadata)
}

func (m *metadataManager) setPinned(alias string, pinned bool) error {
	metadata, err := m.loadAll()
	if err != nil {
		metadata = make(map[string]ServerMetadata)
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
		metadata = make(map[string]ServerMetadata)
	}

	meta := metadata[alias]
	meta.LastSeen = time.Now().Format(time.RFC3339)
	meta.SSHCount++

	metadata[alias] = meta
	return m.saveAll(metadata)
}

func (m *metadataManager) ensureDirectory() error {
	dir := filepath.Dir(m.filePath)
	return os.MkdirAll(dir, 0o750)
}
