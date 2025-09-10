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
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

const (
	MaxBackups     = 10
	TempSuffix     = ".tmp"
	BackupSuffix   = "lazyssh.backup"
	SSHConfigPerms = 0o600
)

// Private methods

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

	if err := r.createBackup(); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	if err := r.fileSystem.Rename(tempFile, r.configPath); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}

	r.logger.Infof("SSH config successfully updated: %s", r.configPath)
	return nil
}

// createTempFile creates a temporary file in the specified directory
func (r *Repository) createTempFile(dir string) (string, error) {
	timestamp := time.Now().Format("20060102150405")
	tempFileName := fmt.Sprintf("config%s%s", timestamp, TempSuffix)
	tempFilePath := filepath.Join(dir, tempFileName)

	// Create the temp file
	file, err := r.fileSystem.Create(tempFilePath)
	if err != nil {
		return "", err
	}

	if err := file.Close(); err != nil {
		r.logger.Warnf("failed to close temporary file %s: %v", tempFilePath, err)
	}

	return tempFilePath, nil
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

// createBackup creates a timestamped backup of the current config file
func (r *Repository) createBackup() error {
	if _, err := r.fileSystem.Stat(r.configPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	backupPath := fmt.Sprintf("%s-%d-%s", r.configPath, timestamp, BackupSuffix)

	if err := r.copyFile(r.configPath, backupPath); err != nil {
		return fmt.Errorf("failed to copy config to backup: %w", err)
	}

	r.logger.Infof("Created backup: %s", backupPath)

	configDir := filepath.Dir(r.configPath)

	backupFiles, err := r.findBackupFiles(configDir)
	if err != nil {
		return err
	}

	if len(backupFiles) <= MaxBackups {
		return nil
	}

	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].ModTime().After(backupFiles[j].ModTime())
	})

	for i := MaxBackups; i < len(backupFiles); i++ {
		backupPath := filepath.Join(configDir, backupFiles[i].Name())
		if err := r.fileSystem.Remove(backupPath); err != nil {
			r.logger.Warnf("failed to remove old backup %s: %v", backupPath, err)
			continue
		}
		r.logger.Infof("Removed old backup: %s", backupPath)
	}
	return nil
}

// copyFile copies a file from src to dst
func (r *Repository) copyFile(src, dst string) error {
	srcFile, err := r.fileSystem.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := srcFile.Close(); cerr != nil {
			r.logger.Warnf("failed to close source file %s: %v", src, cerr)
		}
	}()

	srcInfo, err := r.fileSystem.Stat(src)
	if err != nil {
		return err
	}

	destFile, err := r.fileSystem.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := destFile.Close(); cerr != nil {
			r.logger.Warnf("failed to close destination file %s: %v", dst, cerr)
		}
	}()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// findBackupFiles finds all backup files for the given config file
func (r *Repository) findBackupFiles(dir string) ([]os.FileInfo, error) {
	entries, err := r.fileSystem.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var backupFiles []os.FileInfo

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, BackupSuffix) {
			info, err := entry.Info()
			if err != nil {
				r.logger.Warnf("failed to get info for backup file %s: %v", name, err)
				continue
			}
			backupFiles = append(backupFiles, info)
		}
	}

	return backupFiles, nil
}

// toDomainServer converts ssh_config.Config to a slice of domain.Server.
func (r *Repository) toDomainServer(cfg *ssh_config.Config) []domain.Server {
	servers := make([]domain.Server, 0, len(cfg.Hosts))
	for _, host := range cfg.Hosts {

		aliases := make([]string, 0, len(host.Patterns))

		for _, pattern := range host.Patterns {
			alias := pattern.String()
			// Skip if alias contains wildcards (not a concrete Host)
			if strings.ContainsAny(alias, "!*?[]") {
				continue
			}
			aliases = append(aliases, alias)
		}
		if len(aliases) == 0 {
			continue
		}
		server := domain.Server{
			Alias:         aliases[0],
			Aliases:       aliases,
			Port:          22,
			IdentityFiles: []string{},
		}

		for _, node := range host.Nodes {
			kvNode, ok := node.(*ssh_config.KV)
			if !ok {
				continue
			}

			r.mapKVToServer(&server, kvNode)
		}

		servers = append(servers, server)
	}

	return servers
}

// mapKVToServer maps a ssh_config.KV node to the corresponding fields in domain.Server.
func (r *Repository) mapKVToServer(server *domain.Server, kvNode *ssh_config.KV) {
	switch strings.ToLower(kvNode.Key) {
	case "hostname":
		server.Host = kvNode.Value
	case "user":
		server.User = kvNode.Value
	case "port":
		port, err := strconv.Atoi(kvNode.Value)
		if err == nil {
			server.Port = port
		}
	case "identityfile":
		server.IdentityFiles = append(server.IdentityFiles, kvNode.Value)
	}
}

// mergeMetadata merges additional metadata into the servers.
func (r *Repository) mergeMetadata(servers []domain.Server, metadata map[string]ServerMetadata) []domain.Server {
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

// filterServers filters servers based on the query string.
func (r *Repository) filterServers(servers []domain.Server, query string) []domain.Server {
	query = strings.ToLower(query)
	filtered := make([]domain.Server, 0)

	for _, server := range servers {
		if r.matchesQuery(server, query) {
			filtered = append(filtered, server)
		}
	}

	return filtered
}

// matchesQuery checks if any field of the server matches the query string.
func (r *Repository) matchesQuery(server domain.Server, query string) bool {
	fields := []string{
		strings.ToLower(server.Host),
		strings.ToLower(server.User),
	}
	for _, tag := range server.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	for _, alias := range server.Aliases {
		fields = append(fields, strings.ToLower(alias))
	}

	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

// serverExists checks if a server with the given alias already exists in the config.
func (r *Repository) serverExists(cfg *ssh_config.Config, alias string) bool {
	return r.findHostByAlias(cfg, alias) != nil
}

// findHostByAlias finds a host by its alias in the SSH config.
func (r *Repository) findHostByAlias(cfg *ssh_config.Config, alias string) *ssh_config.Host {
	for _, host := range cfg.Hosts {
		if r.hostContainsPattern(host, alias) {
			return host
		}
	}
	return nil
}

// hostContainsPattern checks if a host contains a specific pattern.
func (r *Repository) hostContainsPattern(host *ssh_config.Host, target string) bool {
	for _, pattern := range host.Patterns {
		if pattern.String() == target {
			return true
		}
	}
	return false
}

// createHostFromServer creates a new ssh_config.Host from a domain.Server.
func (r *Repository) createHostFromServer(server domain.Server) *ssh_config.Host {
	host := &ssh_config.Host{
		Patterns: []*ssh_config.Pattern{
			{Str: server.Alias},
		},
		Nodes:              make([]ssh_config.Node, 0),
		LeadingSpace:       1,
		EOLComment:         "Added by lazyssh",
		SpaceBeforeComment: strings.Repeat(" ", 4),
	}

	r.addKVNodeIfNotEmpty(host, "HostName", server.Host)
	r.addKVNodeIfNotEmpty(host, "User", server.User)
	r.addKVNodeIfNotEmpty(host, "Port", fmt.Sprintf("%d", server.Port))
	for _, identityFile := range server.IdentityFiles {
		r.addKVNodeIfNotEmpty(host, "IdentityFile", identityFile)
	}

	return host
}

// addKVNodeIfNotEmpty adds a key-value node to the host if the value is not empty.
func (r *Repository) addKVNodeIfNotEmpty(host *ssh_config.Host, key, value string) {
	if value == "" {
		return
	}

	kvNode := &ssh_config.KV{
		Key:          key,
		Value:        value,
		LeadingSpace: 4,
	}
	host.Nodes = append(host.Nodes, kvNode)
}

// updateHostNodes updates the nodes of an existing host with new server details.
func (r *Repository) updateHostNodes(host *ssh_config.Host, newServer domain.Server) {
	updates := map[string]string{
		"hostname": newServer.Host,
		"user":     newServer.User,
		"port":     fmt.Sprintf("%d", newServer.Port),
	}
	for key, value := range updates {
		if value != "" {
			r.updateOrAddKVNode(host, key, value)
		}
	}
	// Replace IdentityFile entries entirely to reflect the new state.
	// This ensures removing/clearing identity files works as expected.

	removeKey := func(nodes []ssh_config.Node, key string) []ssh_config.Node {
		filtered := make([]ssh_config.Node, 0, len(nodes))
		for _, node := range nodes {
			if kv, ok := node.(*ssh_config.KV); ok {
				if strings.EqualFold(kv.Key, key) {
					continue // skip existing IdentityFile
				}
			}
			filtered = append(filtered, node)
		}
		return filtered
	}
	host.Nodes = removeKey(host.Nodes, "IdentityFile")

	for _, identityFile := range newServer.IdentityFiles {
		r.addKVNodeIfNotEmpty(host, "IdentityFile", identityFile)
	}
	
}

// updateOrAddKVNode updates an existing key-value node or adds a new one if it doesn't exist.
func (r *Repository) updateOrAddKVNode(host *ssh_config.Host, key, newValue string) {
	keyLower := strings.ToLower(key)

	// Try to update existing node
	for _, node := range host.Nodes {
		kvNode, ok := node.(*ssh_config.KV)
		if ok && strings.EqualFold(kvNode.Key, keyLower) {
			kvNode.Value = newValue
			return
		}
	}

	// Add new node if not found
	kvNode := &ssh_config.KV{
		Key:          r.getProperKeyCase(key),
		Value:        newValue,
		LeadingSpace: 4,
	}
	host.Nodes = append(host.Nodes, kvNode)
}

// getProperKeyCase returns the proper case for known SSH config keys.
// Reference: https://www.ssh.com/academy/ssh/config
func (r *Repository) getProperKeyCase(key string) string {
	keyMap := map[string]string{
		"hostname":     "HostName",
		"user":         "User",
		"port":         "Port",
		"identityfile": "IdentityFile",
	}

	if properCase, exists := keyMap[strings.ToLower(key)]; exists {
		return properCase
	}
	return key
}

// removeHostByAlias removes a host by its alias from the list of hosts.
func (r *Repository) removeHostByAlias(hosts []*ssh_config.Host, alias string) []*ssh_config.Host {
	for i, host := range hosts {
		if r.hostContainsPattern(host, alias) {
			return append(hosts[:i], hosts[i+1:]...)
		}
	}
	return hosts
}
