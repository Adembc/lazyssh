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

	"go.uber.org/zap"
)

func TestLoadConfigWithIncludes(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ssh_config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .ssh directory and subdirectories
	sshDir := filepath.Join(tmpDir, ".ssh")
	configDDir := filepath.Join(sshDir, "config.d")
	if err := os.MkdirAll(configDDir, 0o700); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	// Create included config file
	includedConfigPath := filepath.Join(configDDir, "hosts")
	includedContent := `Host server1
    HostName 192.168.1.1
    User admin
    Port 2222

Host server2
    HostName 192.168.1.2
    User testuser
    Port 3333
`
	if err := os.WriteFile(includedConfigPath, []byte(includedContent), 0o600); err != nil {
		t.Fatalf("failed to write included config: %v", err)
	}

	// Create main config file with Include directive
	mainConfigPath := filepath.Join(sshDir, "config")
	mainContent := `Host server0
    HostName localhost
    User root
    Port 22

Include ~/.ssh/config.d/hosts
`
	if err := os.WriteFile(mainConfigPath, []byte(mainContent), 0o600); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	// Create logger
	logger, _ := zap.NewDevelopment()
	sugared := logger.Sugar()

	// Create repository
	repo := &Repository{
		configPath: mainConfigPath,
		fileSystem: DefaultFileSystem{},
		logger:     sugared,
	}

	// Load config
	cfg, err := repo.loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify that all hosts were loaded
	if len(cfg.Hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(cfg.Hosts))
	}

	// Check that we have the expected hosts
	hostNames := make(map[string]bool)
	for _, host := range cfg.Hosts {
		for _, pattern := range host.Patterns {
			hostNames[pattern.String()] = true
		}
	}

	expectedHosts := []string{"server0", "server1", "server2"}
	for _, expected := range expectedHosts {
		if !hostNames[expected] {
			t.Errorf("expected host %q not found", expected)
		}
	}
}

func TestLoadConfigWithWildcardIncludes(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ssh_config_wildcard_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .ssh directory and subdirectories
	sshDir := filepath.Join(tmpDir, ".ssh")
	configDDir := filepath.Join(sshDir, "config.d")
	if err := os.MkdirAll(configDDir, 0o700); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	// Create multiple included config files
	for i := 1; i <= 2; i++ {
		fileName := filepath.Join(configDDir, "config"+string(rune('0'+i)))
		content := "Host server" + string(rune('0'+i)) + "\n    HostName 192.168.1." + string(rune('0'+i)) + "\n"
		if err := os.WriteFile(fileName, []byte(content), 0o600); err != nil {
			t.Fatalf("failed to write config file: %v", err)
		}
	}

	// Create main config file with wildcard Include directive
	mainConfigPath := filepath.Join(sshDir, "config")
	mainContent := `Host server0
    HostName localhost

Include ~/.ssh/config.d/config*
`
	if err := os.WriteFile(mainConfigPath, []byte(mainContent), 0o600); err != nil {
		t.Fatalf("failed to write main config: %v", err)
	}

	// Create logger
	logger, _ := zap.NewDevelopment()
	sugared := logger.Sugar()

	// Create repository
	repo := &Repository{
		configPath: mainConfigPath,
		fileSystem: DefaultFileSystem{},
		logger:     sugared,
	}

	// Load config
	cfg, err := repo.loadConfig()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify that all hosts were loaded (server0 + server1 + server2)
	if len(cfg.Hosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(cfg.Hosts))
	}
}
