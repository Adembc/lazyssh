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

package ui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

type settingsManager struct {
	filePath string
	logger   *zap.SugaredLogger
}

type uiSettings struct {
	SortMode SortMode `json:"sort_mode,omitempty"`
}

func newSettingsManager(logger *zap.SugaredLogger) *settingsManager {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Warnw("failed to determine home directory for settings", "error", err)
		return nil
	}

	return &settingsManager{
		filePath: filepath.Join(home, ".lazyssh", "settings.json"),
		logger:   logger,
	}
}

func (m *settingsManager) LoadSortMode() (SortMode, error) {
	if m == nil {
		return SortByAliasAsc, errors.New("nil settings manager")
	}

	settings, err := m.load()
	if err != nil {
		return SortByAliasAsc, err
	}

	if !settings.SortMode.valid() {
		return SortByAliasAsc, nil
	}

	return settings.SortMode, nil
}

func (m *settingsManager) SaveSortMode(mode SortMode) error {
	if m == nil {
		return errors.New("nil settings manager")
	}

	settings, err := m.load()
	if err != nil {
		return err
	}

	settings.SortMode = mode
	return m.save(settings)
}

func (m *settingsManager) load() (uiSettings, error) {
	var settings uiSettings

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return settings, err
	}

	if len(data) == 0 {
		return settings, nil
	}

	if err := json.Unmarshal(data, &settings); err != nil {
		// Handle unmarshal errors (e.g., old string format vs new int format)
		// by returning default settings. This allows automatic migration from
		// old format to new format when SaveSortMode is called.
		m.logger.Warnw("failed to parse settings file, using defaults", "error", err, "path", m.filePath)
		return uiSettings{}, nil
	}

	return settings, nil
}

func (m *settingsManager) save(settings uiSettings) error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.filePath, data, 0o600)
}
