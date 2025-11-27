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
	"strings"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

// TestIncludeDirectivesPreservation tests that Include directives are preserved
// when adding, updating, or deleting server entries
func TestIncludeDirectivesPreservation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config")
	metadataPath := filepath.Join(tempDir, "metadata.json")

	// Initial config with Include directives
	initialConfig := `# =====================================================================
# Includes
# =====================================================================
Include terraform.d/*
Include personal.d/*
Include work.d/*
Include ~/.orbstack/ssh/config

# =====================================================================
# Personal Servers
# =====================================================================
Host testserver
    HostName test.example.com
    User testuser
    Port 22
`

	// Write initial config
	err := os.WriteFile(configPath, []byte(initialConfig), 0o600)
	if err != nil {
		t.Fatalf("Failed to write initial config: %v", err)
	}

	// Create repository
	logger := zap.NewNop().Sugar()
	repo := NewRepository(logger, configPath, metadataPath)

	// Test 1: Add a new server
	newServer := domain.Server{
		Alias: "newserver",
		Host:  "new.example.com",
		User:  "newuser",
		Port:  2222,
	}

	err = repo.AddServer(newServer)
	if err != nil {
		t.Fatalf("Failed to add server: %v", err)
	}

	// Read the config file and verify Include directives are preserved
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	configStr := string(content)

	// Check that all Include directives are present
	includeDirectives := []string{
		"Include terraform.d/*",
		"Include personal.d/*",
		"Include work.d/*",
		"Include ~/.orbstack/ssh/config",
	}

	for _, directive := range includeDirectives {
		if !strings.Contains(configStr, directive) {
			t.Errorf("Include directive missing after AddServer: %s\nConfig content:\n%s", directive, configStr)
		}
	}

	// Verify the new server was added
	if !strings.Contains(configStr, "Host newserver") {
		t.Errorf("New server not found in config")
	}

	// Test 2: Update existing server
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatalf("Failed to get servers: %v", err)
	}

	var testServer domain.Server
	for _, s := range servers {
		if s.Alias == "testserver" {
			testServer = s
			break
		}
	}

	updatedServer := testServer
	updatedServer.Port = 2200
	err = repo.UpdateServer(testServer, updatedServer)
	if err != nil {
		t.Fatalf("Failed to update server: %v", err)
	}

	// Read config again
	content, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	configStr = string(content)

	// Check that Include directives are still present
	for _, directive := range includeDirectives {
		if !strings.Contains(configStr, directive) {
			t.Errorf("Include directive missing after UpdateServer: %s\nConfig content:\n%s", directive, configStr)
		}
	}

	// Test 3: Delete a server
	err = repo.DeleteServer(newServer)
	if err != nil {
		t.Fatalf("Failed to delete server: %v", err)
	}

	// Read config again
	content, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	configStr = string(content)

	// Check that Include directives are still present
	for _, directive := range includeDirectives {
		if !strings.Contains(configStr, directive) {
			t.Errorf("Include directive missing after DeleteServer: %s\nConfig content:\n%s", directive, configStr)
		}
	}

	// Verify the server was deleted
	if strings.Contains(configStr, "Host newserver") {
		t.Errorf("Deleted server still present in config")
	}
}
