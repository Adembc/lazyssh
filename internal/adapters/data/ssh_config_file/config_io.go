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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
)

// loadConfig reads and parses the SSH config file, including any files specified by Include directives.
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

	// Process include directives to merge hosts from included files
	if err := r.processIncludes(cfg, r.configPath); err != nil {
		r.logger.Warnf("error processing include directives: %v", err)
		// Continue without the included files rather than failing completely
	}

	return cfg, nil
}

// processIncludes recursively processes Include directives in the config and merges hosts from included files.
// It tracks visited files to prevent infinite loops.
func (r *Repository) processIncludes(cfg *ssh_config.Config, configPath string) error {
	visited := make(map[string]bool)
	includePatterns, err := r.extractIncludePatterns(configPath)
	if err != nil {
		return err
	}

	return r.processIncludePatterns(cfg, configPath, includePatterns, visited, 0)
}

// extractIncludePatterns extracts all Include directive patterns from a config file
func (r *Repository) extractIncludePatterns(filePath string) ([]string, error) {
	file, err := r.fileSystem.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close file: %v", cerr)
		}
	}()

	var patterns []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if line starts with "Include" (case-insensitive)
		if strings.HasPrefix(strings.ToLower(line), "include") {
			// Parse the include line to extract the pattern(s)
			// Format: Include /path/to/file [/another/path/to/file] ...
			parts := strings.Fields(line)
			if len(parts) > 1 {
				// Add all patterns after "Include"
				for i := 1; i < len(parts); i++ {
					patterns = append(patterns, parts[i])
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return patterns, nil
}

// processIncludePatterns recursively processes include patterns with depth limit to prevent infinite recursion.
func (r *Repository) processIncludePatterns(cfg *ssh_config.Config, configPath string, patterns []string, visited map[string]bool, depth uint8) error {
	const maxDepth = 5

	if depth > maxDepth {
		return fmt.Errorf("include nesting depth exceeded (max: %d)", maxDepth)
	}

	configDir := filepath.Dir(configPath)

	for _, pattern := range patterns {
		includedHosts, err := r.loadIncludedConfig(pattern, configDir, visited, depth+1)
		if err != nil {
			r.logger.Warnf("failed to process include pattern %s: %v", pattern, err)
			continue
		}

		// Merge hosts from included file into main config
		cfg.Hosts = append(cfg.Hosts, includedHosts...)
	}

	return nil
}

// loadIncludedConfig loads and parses a single included config file and any of its includes
func (r *Repository) loadIncludedConfig(pattern, baseDir string, visited map[string]bool, depth uint8) ([]*ssh_config.Host, error) {
	var includePath string

	// Handle tilde expansion for home directory
	if strings.HasPrefix(pattern, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		includePath = filepath.Join(home, pattern[1:])
	} else if filepath.IsAbs(pattern) {
		// Resolve the include path
		includePath = pattern
	} else {
		includePath = filepath.Join(baseDir, pattern)
	}

	// Handle wildcards
	matches, err := filepath.Glob(includePath)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %s: %w", includePath, err)
	}

	var allHosts []*ssh_config.Host

	for _, match := range matches {
		// Check if already visited (prevent cycles)
		absPath, _ := filepath.Abs(match)
		if visited[absPath] {
			continue
		}
		visited[absPath] = true

		file, err := r.fileSystem.Open(match)
		if err != nil {
			r.logger.Warnf("failed to open included config file %s: %v", match, err)
			continue
		}

		includedCfg, err := ssh_config.Decode(file)
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close file %s: %v", match, cerr)
		}

		if err != nil {
			r.logger.Warnf("failed to decode included config file %s: %v", match, err)
			continue
		}

		// Recursively process any includes in the included file
		nestedPatterns, err := r.extractIncludePatterns(match)
		if err != nil {
			r.logger.Warnf("error extracting includes from %s: %v", match, err)
		} else if len(nestedPatterns) > 0 {
			if err := r.processIncludePatterns(includedCfg, match, nestedPatterns, visited, depth+1); err != nil {
				r.logger.Warnf("error processing nested includes in %s: %v", match, err)
			}
		}

		allHosts = append(allHosts, includedCfg.Hosts...)
	}

	return allHosts, nil
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
