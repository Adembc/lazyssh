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
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_NonExistentFile(t *testing.T) {
	settings, err := LoadSettings("/nonexistent/path/metadata.json")
	if err != nil {
		t.Errorf("LoadSettings() with non-existent file should not error, got %v", err)
	}
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with non-existent file should return empty theme, got %q", settings.Theme)
	}
}

func TestLoadSettings_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	if err := os.WriteFile(tmpFile, []byte{}, 0o644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Errorf("LoadSettings() with empty file should not error, got %v", err)
	}
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with empty file should return empty theme, got %q", settings.Theme)
	}
}

func TestLoadSettings_NewFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	content := `{
		"settings": {"theme": "light"},
		"servers": {"server1": {"tags": ["prod"]}}
	}`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Errorf("LoadSettings() unexpected error: %v", err)
	}
	if settings.Theme != "light" {
		t.Errorf("LoadSettings() theme = %q, want %q", settings.Theme, "light")
	}
}

func TestLoadSettings_OldFormat(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	// Old format: servers directly at root level
	content := `{
		"server1": {"tags": ["prod"], "ssh_count": 5},
		"server2": {"tags": ["dev"]}
	}`
	if err := os.WriteFile(tmpFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	settings, err := LoadSettings(tmpFile)
	if err != nil {
		t.Errorf("LoadSettings() unexpected error: %v", err)
	}
	// Old format has no settings, should return empty theme
	if settings.Theme != "" {
		t.Errorf("LoadSettings() with old format should return empty theme, got %q", settings.Theme)
	}
}

func TestMetadataManager_SaveAndGetSettings(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	mm := newMetadataManager(tmpFile, nil)

	// Save settings
	settings := Settings{Theme: "light"}
	if err := mm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() unexpected error: %v", err)
	}

	// Read back
	got, err := mm.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() unexpected error: %v", err)
	}
	if got.Theme != "light" {
		t.Errorf("GetSettings() theme = %q, want %q", got.Theme, "light")
	}
}

func TestMetadataManager_SettingsPreserveServers(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	mm := newMetadataManager(tmpFile, nil)

	// First, save some server metadata
	serverMeta := map[string]ServerMetadata{
		"server1": {Tags: []string{"prod"}, SSHCount: 5},
	}
	if err := mm.saveAll(serverMeta); err != nil {
		t.Fatalf("saveAll() unexpected error: %v", err)
	}

	// Now save settings
	settings := Settings{Theme: "light"}
	if err := mm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() unexpected error: %v", err)
	}

	// Verify servers are still there
	servers, err := mm.loadAll()
	if err != nil {
		t.Fatalf("loadAll() unexpected error: %v", err)
	}
	if len(servers) != 1 {
		t.Errorf("Expected 1 server, got %d", len(servers))
	}
	if servers["server1"].SSHCount != 5 {
		t.Errorf("Server metadata was not preserved, SSHCount = %d, want 5", servers["server1"].SSHCount)
	}

	// Verify settings are there too
	got, err := mm.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() unexpected error: %v", err)
	}
	if got.Theme != "light" {
		t.Errorf("GetSettings() theme = %q, want %q", got.Theme, "light")
	}
}

func TestMetadataManager_ServersSavePreservesSettings(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "metadata.json")

	mm := newMetadataManager(tmpFile, nil)

	// First, save settings
	settings := Settings{Theme: "light"}
	if err := mm.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings() unexpected error: %v", err)
	}

	// Now save server metadata
	serverMeta := map[string]ServerMetadata{
		"server1": {Tags: []string{"prod"}, SSHCount: 10},
	}
	if err := mm.saveAll(serverMeta); err != nil {
		t.Fatalf("saveAll() unexpected error: %v", err)
	}

	// Verify settings are still there
	got, err := mm.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() unexpected error: %v", err)
	}
	if got.Theme != "light" {
		t.Errorf("Settings were not preserved, theme = %q, want %q", got.Theme, "light")
	}

	// Verify servers are there too
	servers, err := mm.loadAll()
	if err != nil {
		t.Fatalf("loadAll() unexpected error: %v", err)
	}
	if servers["server1"].SSHCount != 10 {
		t.Errorf("Server metadata incorrect, SSHCount = %d, want 10", servers["server1"].SSHCount)
	}
}
