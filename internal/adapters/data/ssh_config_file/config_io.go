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
	"os"
	"path/filepath"
	"time"

	"github.com/kevinburke/ssh_config"
)

// loadConfig reads and parses the SSH config file.
// If the file does not exist, it returns an empty config without error to support first-run behavior.
func (r *Repository) loadConfig() (*ssh_config.Config, error) {
	file, err := r.fileSystem.Open(r.configPath)
	if err != nil {
		if r.fileSystem.IsNotExist(err) {
			return &ssh_config.Config{Hosts: []*ssh_config.Host{}}, nil
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close config file: %v", cerr)
		}
	}()

	cfg, err := ssh_config.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return cfg, nil
}

// saveConfig writes the SSH config back to the file with atomic operations and backup management.
func (r *Repository) saveConfig(cfg *ssh_config.Config) error {
	configDir := filepath.Dir(r.configPath)

	tempFile, err := r.createTempFile(configDir)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer func() {
		if removeErr := r.fileSystem.Remove(tempFile); removeErr != nil {
			r.logger.Warnf("failed to remove temporary file %s: %v", tempFile, removeErr)
		}
	}()

	if err := r.writeConfigToFile(tempFile, cfg); err != nil {
		return fmt.Errorf("failed to write config to temporary file: %w", err)
	}

	// Ensure a one-time original backup exists before any modifications managed by lazyssh.
	if err := r.createOriginalBackupIfNeeded(); err != nil {
		return fmt.Errorf("failed to create original backup: %w", err)
	}

	if err := r.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := r.fileSystem.Rename(tempFile, r.configPath); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}

	r.logger.Infof("SSH config successfully updated: %s", r.configPath)
	return nil
}

// writeConfigToFile writes the SSH config content to the specified file
func (r *Repository) writeConfigToFile(filePath string, cfg *ssh_config.Config) error {
	file, err := r.fileSystem.OpenFile(filePath, os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close file %s: %v", filePath, cerr)
		}
	}()

	configContent := cfg.String()
	if _, err := file.WriteString(configContent); err != nil {
		return fmt.Errorf("failed to write config content: %w", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file to disk: %w", err)
	}

	return nil
}

// createTempFile creates a temporary file in the specified directory
func (r *Repository) createTempFile(dir string) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	tempFileName := fmt.Sprintf("config%s%s", timestamp, TempSuffix)
	tempFilePath := filepath.Join(dir, tempFileName)

	// Create the temp file with explicit 0600 permissions
	f, err := r.fileSystem.OpenFile(tempFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, SSHConfigPerms)
	if err != nil {
		return "", err
	}
	if cerr := f.Close(); cerr != nil {
		r.logger.Warnf("failed to close temporary file %s: %v", tempFilePath, cerr)
	}

	return tempFilePath, nil
}
