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

package services

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"go.uber.org/zap"
)

const (
	// ScopeLocal represents repository-level git configuration.
	ScopeLocal = "local"
	// ScopeGlobal represents user-level git configuration.
	ScopeGlobal = "global"
	// ScopeBoth represents both repository and user-level git configuration.
	ScopeBoth = "both"

	// SSH key type constants
	keyTypeRSA     = "rsa"
	keyTypeEd25519 = "ed25519"
	keyTypeECDSA   = "ecdsa"
	keyTypeDSA     = "dsa"

	// SSH file constants
	fileKnownHosts     = "known_hosts"
	fileConfig         = "config"
	fileAuthorizedKeys = "authorized_keys"
)

type gitService struct {
	logger           *zap.SugaredLogger
	serverRepository ports.ServerRepository
}

// NewGitService creates a new instance of gitService.
func NewGitService(logger *zap.SugaredLogger) ports.GitService {
	return &gitService{
		logger: logger,
	}
}

// SetServerRepository sets the server repository for the git service.
// This is needed to resolve SSH key paths from SSH config.
func (gs *gitService) SetServerRepository(repo ports.ServerRepository) {
	gs.serverRepository = repo
}

// IsGitRepository checks if the given path is inside a git repository.
func (gs *gitService) IsGitRepository(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetGitRootPath returns the root path of the git repository.
func (gs *gitService) GetGitRootPath(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository or git command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetPushRemoteURL returns the push remote name and URL for the git repository.
// It searches for remotes with push URLs, preferring "origin" if available.
func (gs *gitService) GetPushRemoteURL(repoPath string) (remoteName, remoteURL string, err error) {
	// First try to get origin push URL
	cmd := exec.Command("git", "-C", repoPath, "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to get git remotes: %w", err)
	}

	// Parse git remote -v output
	// Format: "origin    git@github.com:user/repo.git (fetch)"
	//         "origin    git@github.com:user/repo.git (push)"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	var otherRemoteName, otherPushURL string

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		name := parts[0]
		url := parts[1]
		operation := strings.TrimPrefix(strings.TrimSuffix(parts[2], ")"), "(")

		if operation == "push" {
			if name == "origin" {
				remoteName = name
				remoteURL = url
				return remoteName, remoteURL, nil
			} else if otherRemoteName == "" {
				otherRemoteName = name
				otherPushURL = url
			}
		}
	}

	// Prefer origin, otherwise use the first available push remote
	if otherRemoteName != "" {
		return otherRemoteName, otherPushURL, nil
	}

	return "", "", fmt.Errorf("no push remote found")
}

// ListSSHKeys lists all SSH private keys from the SSH config and the specified SSH directory.
func (gs *gitService) ListSSHKeys(sshDir string, serverRepo ports.ServerRepository) ([]ports.SSHKey, error) {
	keyMap := make(map[string]*ports.SSHKey)

	// First, get keys from SSH config (this is the primary source)
	gs.addKeysFromSSHConfig(serverRepo, keyMap)

	// Also scan ~/.ssh/ directory for additional keys
	if sshDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		sshDir = filepath.Join(homeDir, ".ssh")
	}

	if err := gs.addKeysFromDirectory(sshDir, keyMap); err != nil && len(keyMap) == 0 {
		return nil, err
	}

	// Check which keys are loaded in ssh-agent/keychain
	gs.markKeysInAgent(keyMap)

	return gs.keysMapToSlice(keyMap), nil
}

func (gs *gitService) addKeysFromSSHConfig(serverRepo ports.ServerRepository, keyMap map[string]*ports.SSHKey) {
	if serverRepo == nil {
		return
	}

	servers, err := serverRepo.ListServers("")
	if err != nil {
		return
	}

	for _, server := range servers {
		for _, identityFile := range server.IdentityFiles {
			gs.addKeyFromPath(identityFile, keyMap)
		}
	}
}

func (gs *gitService) addKeyFromPath(identityFile string, keyMap map[string]*ports.SSHKey) {
	if identityFile == "" {
		return
	}

	// Expand ~ to home directory
	if strings.HasPrefix(identityFile, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			identityFile = filepath.Join(homeDir, identityFile[2:])
		}
	}

	// Check if key already exists in map
	if _, exists := keyMap[identityFile]; exists {
		return
	}

	// Check if file exists
	info, err := os.Stat(identityFile)
	if err != nil {
		gs.logger.Debugf("Identity file not found: %s", identityFile)
		return
	}

	isPrivateKey, isEncrypted := gs.isSSHPrivateKey(identityFile)
	if !isPrivateKey {
		return
	}

	key := &ports.SSHKey{
		Name:        filepath.Base(identityFile),
		Path:        identityFile,
		HasPubKey:   false,
		ModTime:     info.ModTime(),
		IsEncrypted: isEncrypted,
	}

	// Check for corresponding public key
	pubKeyPath := identityFile + ".pub"
	if _, err := os.Stat(pubKeyPath); err == nil {
		key.HasPubKey = true
	}

	keyMap[identityFile] = key
}

func (gs *gitService) addKeysFromDirectory(sshDir string, keyMap map[string]*ports.SSHKey) error {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if len(keyMap) > 0 {
			gs.logger.Debugf("Could not read SSH directory %s: %v", sshDir, err)
			return nil
		}
		return fmt.Errorf("failed to read SSH directory: %w", err)
	}

	// Process files from directory
	for _, entry := range entries {
		if entry.IsDir() || gs.shouldSkipFile(entry.Name()) {
			continue
		}

		keyPath := filepath.Join(sshDir, entry.Name())
		if _, exists := keyMap[keyPath]; exists {
			continue
		}

		gs.addKeyFromDirectoryEntry(entry, sshDir, keyMap)
	}

	// Check for corresponding public keys
	gs.markPublicKeys(entries, sshDir, keyMap)
	return nil
}

func (gs *gitService) shouldSkipFile(name string) bool {
	return name == fileKnownHosts || name == fileConfig || name == fileAuthorizedKeys ||
		strings.HasSuffix(name, ".pub") || strings.HasPrefix(name, ".")
}

func (gs *gitService) addKeyFromDirectoryEntry(entry os.DirEntry, sshDir string, keyMap map[string]*ports.SSHKey) {
	keyPath := filepath.Join(sshDir, entry.Name())

	info, err := entry.Info()
	if err != nil {
		gs.logger.Warnf("Failed to get info for %s: %v", keyPath, err)
		return
	}

	isPrivateKey, isEncrypted := gs.isSSHPrivateKey(keyPath)
	if !isPrivateKey {
		return
	}

	key := &ports.SSHKey{
		Name:        entry.Name(),
		Path:        keyPath,
		HasPubKey:   false,
		ModTime:     info.ModTime(),
		IsEncrypted: isEncrypted,
	}

	keyMap[keyPath] = key
}

func (gs *gitService) markPublicKeys(entries []os.DirEntry, sshDir string, keyMap map[string]*ports.SSHKey) {
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pub") {
			baseName := strings.TrimSuffix(entry.Name(), ".pub")
			baseKeyPath := filepath.Join(sshDir, baseName)
			if key, exists := keyMap[baseKeyPath]; exists {
				key.HasPubKey = true
			}
		}
	}
}

func (gs *gitService) markKeysInAgent(keyMap map[string]*ports.SSHKey) {
	// Get list of keys loaded in ssh-agent
	agentKeys, err := gs.GetLoadedAgentKeys()
	if err != nil {
		gs.logger.Debugf("Could not check ssh-agent keys: %v", err)
		return
	}

	if len(agentKeys) == 0 {
		return
	}

	// For each key in our map, check if it's in the agent
	for keyPath, key := range keyMap {
		// Check if any agent key line contains this key path
		for _, agentLine := range agentKeys {
			if strings.Contains(agentLine, keyPath) || strings.Contains(agentLine, key.Name) {
				key.InAgent = true
				break
			}
		}
	}
}

func (gs *gitService) keysMapToSlice(keyMap map[string]*ports.SSHKey) []ports.SSHKey {
	keys := make([]ports.SSHKey, 0, len(keyMap))
	for _, key := range keyMap {
		keys = append(keys, *key)
	}

	// Sort by: 1) keys in agent first, 2) then by modification time (newest first)
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].InAgent != keys[j].InAgent {
			return keys[i].InAgent // Keys in agent come first
		}
		return keys[i].ModTime.After(keys[j].ModTime)
	})

	return keys
}

// isSSHPrivateKey checks if a file is an SSH private key and if it's encrypted.
func (gs *gitService) isSSHPrivateKey(path string) (bool, bool) {
	//nolint:gosec // Reading SSH keys from user's .ssh directory is intentional
	file, err := os.Open(path)
	if err != nil {
		return false, false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			gs.logger.Warnf("Failed to close file %s: %v", path, closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	isPrivateKey := false
	isEncrypted := false

	// Read first few lines
	lineCount := 0
	for scanner.Scan() && lineCount < 10 {
		line := scanner.Text()
		lineCount++

		if strings.Contains(line, "BEGIN") && strings.Contains(line, "PRIVATE KEY") {
			isPrivateKey = true
		}

		if strings.Contains(line, "ENCRYPTED") {
			isEncrypted = true
		}

		// Old OpenSSH format check
		if strings.Contains(line, "Proc-Type: 4,ENCRYPTED") {
			isEncrypted = true
		}
	}

	return isPrivateKey, isEncrypted
}

// ConfigureGitSSHKey configures Git to use the specified SSH key.
// scope can be "local" (repository-level) or "global" (user-level).
func (gs *gitService) ConfigureGitSSHKey(repoPath string, keyPath string, scope string) error {
	if scope != ScopeLocal && scope != ScopeGlobal {
		return fmt.Errorf("invalid scope: %s (must be 'local' or 'global')", scope)
	}

	// Verify the key file exists
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("SSH key not found at %s: %w", keyPath, err)
	}

	// Build the SSH command
	sshCommand := fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes", keyPath)

	var cmd *exec.Cmd
	if scope == ScopeLocal {
		// Repository-level configuration
		//nolint:gosec // git command with controlled arguments is safe
		cmd = exec.Command("git", "-C", repoPath, "config", "--local", "core.sshCommand", sshCommand)
	} else {
		// Global configuration
		//nolint:gosec // git command with controlled arguments is safe
		cmd = exec.Command("git", "config", "--global", "core.sshCommand", sshCommand)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to configure Git SSH key: %w\nOutput: %s", err, string(output))
	}

	gs.logger.Infof("Successfully configured Git to use SSH key: %s (scope: %s)", keyPath, scope)
	return nil
}

// GetCurrentGitSSHConfig retrieves the current Git SSH configuration.
func (gs *gitService) GetCurrentGitSSHConfig(repoPath string) (string, error) {
	// Try local config first
	cmd := exec.Command("git", "-C", repoPath, "config", "--local", "core.sshCommand")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	// Try global config
	cmd = exec.Command("git", "config", "--global", "core.sshCommand")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	// Try GIT_SSH_COMMAND environment variable
	envSSH := os.Getenv("GIT_SSH_COMMAND")
	if envSSH != "" {
		return envSSH + " (from environment)", nil
	}

	return "", nil
}

// ClearGitSSHConfig removes the Git SSH configuration, resetting to default behavior.
// scope can be "local" (repository-level), "global" (user-level), or "both".
func (gs *gitService) ClearGitSSHConfig(repoPath string, scope string) error {
	if scope != ScopeLocal && scope != ScopeGlobal && scope != ScopeBoth {
		return fmt.Errorf("invalid scope: %s (must be 'local', 'global', or 'both')", scope)
	}

	var errs []string

	// Clear local configuration
	if scope == ScopeLocal || scope == ScopeBoth {
		cmd := exec.Command("git", "-C", repoPath, "config", "--local", "--unset", "core.sshCommand")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Exit code 5 means key not found, which is not an error for unset
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 5 {
				errs = append(errs, fmt.Sprintf("local: %v (output: %s)", err, string(output)))
			}
		} else {
			gs.logger.Infof("Cleared local Git SSH configuration for %s", repoPath)
		}
	}

	// Clear global configuration
	if scope == ScopeGlobal || scope == ScopeBoth {
		cmd := exec.Command("git", "config", "--global", "--unset", "core.sshCommand")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// Exit code 5 means key not found, which is not an error for unset
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 5 {
				errs = append(errs, fmt.Sprintf("global: %v (output: %s)", err, string(output)))
			}
		} else {
			gs.logger.Infof("Cleared global Git SSH configuration")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to clear Git SSH config: %s", strings.Join(errs, "; "))
	}

	return nil
}

// GetLoadedAgentKeys returns a list of SSH key fingerprints currently loaded in ssh-agent/keychain.
func (gs *gitService) GetLoadedAgentKeys() ([]string, error) {
	cmd := exec.Command("ssh-add", "-l")
	output, err := cmd.Output()
	if err != nil {
		// Exit code 1 means no keys loaded, which is not an error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list ssh-agent keys: %w", err)
	}

	var fingerprints []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Format: "4096 SHA256:... comment (RSA)"
		// We want to extract the file path if available, or the comment
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// The fingerprint is in parts[1], but we also want to capture the full line
			// to match against key paths later
			fingerprints = append(fingerprints, line)
		}
	}

	return fingerprints, nil
}

// ListAllSSHKeys returns all SSH keys from config identity files and ssh-agent.
func (gs *gitService) ListAllSSHKeys(serverRepo ports.ServerRepository) ([]domain.SSHKey, error) {
	keysMap := make(map[string]*domain.SSHKey)

	// Get keys from ssh-agent first to know which are loaded
	agentKeys := make(map[string]agentKeyInfo)
	cmd := exec.Command("ssh-add", "-l")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			// Parse: "4096 SHA256:5NUhY... comment (RSA)"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info := agentKeyInfo{
					fingerprint: parts[1],
					comment:     "",
				}
				if len(parts) >= 3 {
					endIdx := len(parts) - 1
					switch {
					case strings.HasSuffix(line, "(RSA)"):
						endIdx = len(parts) - 1
					case strings.HasSuffix(line, "(ED25519)"):
						endIdx = len(parts) - 1
					case strings.HasSuffix(line, "(ECDSA)"):
						endIdx = len(parts) - 1
					}
					info.comment = strings.Join(parts[2:endIdx], " ")
				}
				// Use fingerprint as key since we may not have path yet
				agentKeys[parts[1]] = info
			}
		}
	}

	// Get identity files from SSH config
	servers, _ := serverRepo.ListServers("")
	for _, server := range servers {
		for _, identityFile := range server.IdentityFiles {
			expandedPath := identityFile
			if strings.HasPrefix(identityFile, "~/") {
				home, _ := os.UserHomeDir()
				expandedPath = filepath.Join(home, identityFile[2:])
			}

			if _, exists := keysMap[expandedPath]; !exists {
				if key := gs.parseKeyFile(expandedPath, agentKeys); key != nil {
					keysMap[expandedPath] = key
				}
			}
		}
	}

	// Also scan ~/.ssh/ directory
	home, err := os.UserHomeDir()
	if err == nil {
		sshDir := filepath.Join(home, ".ssh")
		if entries, err := os.ReadDir(sshDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				// Skip public keys, known_hosts, config, etc.
				if strings.HasSuffix(name, ".pub") || name == fileKnownHosts ||
					name == fileConfig || name == fileAuthorizedKeys {
					continue
				}

				fullPath := filepath.Join(sshDir, name)
				if _, exists := keysMap[fullPath]; !exists {
					if key := gs.parseKeyFile(fullPath, agentKeys); key != nil {
						keysMap[fullPath] = key
					}
				}
			}
		}
	}

	// Convert map to slice
	keys := make([]domain.SSHKey, 0, len(keysMap))
	for _, key := range keysMap {
		keys = append(keys, *key)
	}

	// Sort: loaded keys first, then by name
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].LoadedInAgent != keys[j].LoadedInAgent {
			return keys[i].LoadedInAgent
		}
		return keys[i].Name < keys[j].Name
	})

	return keys, nil
}

type agentKeyInfo struct {
	fingerprint string
	comment     string
}

func (gs *gitService) parseKeyFile(path string, agentKeys map[string]agentKeyInfo) *domain.SSHKey {
	// Check if file exists and is readable
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// Read first few bytes to detect key type
	// #nosec G304 -- path is validated from SSH config or ~/.ssh directory
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Must start with proper SSH key header
	contentStr := string(content)
	if !strings.Contains(contentStr, "PRIVATE KEY") {
		return nil
	}

	key := &domain.SSHKey{
		Path:          path,
		Name:          filepath.Base(path),
		ModTime:       info.ModTime(),
		Source:        "config",
		LoadedInAgent: false,
	}

	// Detect key type and encryption
	switch {
	case strings.Contains(contentStr, "RSA PRIVATE KEY"):
		key.Type = keyTypeRSA
		key.IsEncrypted = strings.Contains(contentStr, "ENCRYPTED")
	case strings.Contains(contentStr, "OPENSSH PRIVATE KEY"):
		// Modern format - could be RSA, Ed25519, ECDSA
		lowerName := strings.ToLower(key.Name)
		switch {
		case strings.Contains(path, "ed25519") || strings.Contains(lowerName, "ed25519"):
			key.Type = keyTypeEd25519
		case strings.Contains(path, "ecdsa") || strings.Contains(lowerName, "ecdsa"):
			key.Type = keyTypeECDSA
		default:
			key.Type = keyTypeRSA // Default assumption
		}
		// Check for encryption by looking for bcrypt cipher indicator
		// Modern OpenSSH keys use "bcrypt" for encryption (base64: YmNyeXB0)
		// Old PEM format uses "Proc-Type: 4,ENCRYPTED"
		key.IsEncrypted = strings.Contains(contentStr, "Proc-Type: 4,ENCRYPTED") ||
			strings.Contains(contentStr, "YmNyeXB0") // bcrypt cipher indicator
	case strings.Contains(contentStr, "DSA PRIVATE KEY"):
		key.Type = keyTypeDSA
		key.IsEncrypted = strings.Contains(contentStr, "ENCRYPTED")
	case strings.Contains(contentStr, "EC PRIVATE KEY"):
		key.Type = keyTypeECDSA
		key.IsEncrypted = strings.Contains(contentStr, "ENCRYPTED")
	default:
		return nil // Unknown key type
	}

	// Check for public key
	pubPath := path + ".pub"
	if _, err := os.Stat(pubPath); err == nil {
		key.HasPublicKey = true

		// Read public key for comment and fingerprint
		// #nosec G304 -- pubPath is derived from validated private key path
		if pubContent, err := os.ReadFile(pubPath); err == nil {
			pubKeyLine := strings.TrimSpace(string(pubContent))
			parts := strings.Fields(pubKeyLine)
			if len(parts) >= 3 {
				key.Comment = strings.Join(parts[2:], " ")
			}
			// Store the full public key line for unloading (same format as ssh-add -L output)
			key.PublicKeyLine = pubKeyLine
		}

		// Get fingerprint using ssh-keygen
		// #nosec G204 -- pubPath is derived from validated private key path
		cmd := exec.Command("ssh-keygen", "-lf", pubPath)
		if output, err := cmd.Output(); err == nil {
			parts := strings.Fields(string(output))
			if len(parts) >= 2 {
				key.Size, _ = strconv.Atoi(parts[0])
				key.Fingerprint = parts[1]

				// Check if this key is loaded in agent
				for fp, info := range agentKeys {
					if fp == key.Fingerprint || info.comment == key.Comment {
						key.LoadedInAgent = true
						if key.Comment == "" {
							key.Comment = info.comment
						}
						break
					}
				}
			}
		}
	}

	return key
}

// ListSSHKeysFromConfig returns SSH keys from config files (IdentityFile entries) and ~/.ssh directory.
func (gs *gitService) ListSSHKeysFromConfig(serverRepo ports.ServerRepository) ([]domain.SSHKey, error) {
	keysMap := make(map[string]*domain.SSHKey)

	// Get keys from ssh-agent to mark which ones are loaded
	agentKeys := make(map[string]agentKeyInfo)
	cmd := exec.Command("ssh-add", "-l")
	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info := agentKeyInfo{
					fingerprint: parts[1],
					comment:     "",
				}
				if len(parts) >= 3 {
					endIdx := len(parts) - 1
					switch {
					case strings.HasSuffix(line, "(RSA)"):
						endIdx = len(parts) - 1
					case strings.HasSuffix(line, "(ED25519)"):
						endIdx = len(parts) - 1
					case strings.HasSuffix(line, "(ECDSA)"):
						endIdx = len(parts) - 1
					}
					info.comment = strings.Join(parts[2:endIdx], " ")
				}
				agentKeys[parts[1]] = info
			}
		}
	}

	// Get identity files from SSH config
	if serverRepo != nil {
		servers, _ := serverRepo.ListServers("")
		for _, server := range servers {
			for _, identityFile := range server.IdentityFiles {
				expandedPath := identityFile
				if strings.HasPrefix(identityFile, "~/") {
					home, _ := os.UserHomeDir()
					expandedPath = filepath.Join(home, identityFile[2:])
				}

				if _, exists := keysMap[expandedPath]; !exists {
					if key := gs.parseKeyFile(expandedPath, agentKeys); key != nil {
						key.Source = "config"
						keysMap[expandedPath] = key
					}
				}
			}
		}
	}

	// Also scan ~/.ssh/ directory
	home, err := os.UserHomeDir()
	if err == nil {
		sshDir := filepath.Join(home, ".ssh")
		if entries, err := os.ReadDir(sshDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := entry.Name()
				if strings.HasSuffix(name, ".pub") || name == fileKnownHosts ||
					name == fileConfig || name == fileAuthorizedKeys {
					continue
				}

				fullPath := filepath.Join(sshDir, name)
				if _, exists := keysMap[fullPath]; !exists {
					if key := gs.parseKeyFile(fullPath, agentKeys); key != nil {
						key.Source = "filesystem"
						keysMap[fullPath] = key
					}
				}
			}
		}
	}

	// Convert map to slice
	keys := make([]domain.SSHKey, 0, len(keysMap))
	for _, key := range keysMap {
		keys = append(keys, *key)
	}

	// Sort by name
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name < keys[j].Name
	})

	return keys, nil
}

// ListSSHKeysFromAgent returns SSH keys currently loaded in ssh-agent,
// merged with their file system information to enable comment editing.
func (gs *gitService) ListSSHKeysFromAgent() ([]domain.SSHKey, error) {
	keys := make([]domain.SSHKey, 0)

	// Get keys from ssh-agent using -L (uppercase L) to get full public key lines
	cmd := exec.Command("ssh-add", "-L")
	output, err := cmd.Output()
	if err != nil {
		// ssh-add returns error if agent has no keys
		return keys, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		key := gs.parseAgentKeyLine(line)
		if key != nil {
			keys = append(keys, *key)
		}
	}

	// Merge with filesystem keys to populate Path field
	keys = gs.mergeAgentKeysWithFilesystem(keys)

	return keys, nil
}

// parseAgentKeyLine parses a single line from ssh-add -L output.
func (gs *gitService) parseAgentKeyLine(line string) *domain.SSHKey {
	if line == "" {
		return nil
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return nil
	}

	key := &domain.SSHKey{
		LoadedInAgent: true,
		PublicKeyLine: line,
		Source:        "agent",
	}

	// Extract key type from first field
	keyType := strings.TrimPrefix(parts[0], "ssh-")
	switch keyType {
	case "rsa":
		key.Type = keyTypeRSA
	case "ed25519":
		key.Type = keyTypeEd25519
	case "ecdsa":
		key.Type = keyTypeECDSA
	case "dsa":
		key.Type = keyTypeDSA
	default:
		// Try to detect from other formats
		switch {
		case strings.Contains(line, "RSA"):
			key.Type = keyTypeRSA
		case strings.Contains(line, "ED25519"):
			key.Type = keyTypeEd25519
		case strings.Contains(line, "ECDSA"):
			key.Type = keyTypeECDSA
		}
	}

	// Extract comment (everything after the key data)
	if len(parts) >= 3 {
		key.Comment = strings.Join(parts[2:], " ")
		key.Name = key.Comment
	} else {
		// No comment, use a truncated version of the key data as name
		if len(parts[1]) > 16 {
			key.Name = parts[1][:16] + "..."
		} else {
			key.Name = parts[1]
		}
	}

	// Compute fingerprint from the public key for reliable matching
	// Use ssh-keygen -lf - to compute fingerprint from stdin
	var stdout, stderr bytes.Buffer
	// #nosec G204 -- line is from trusted ssh-add output
	cmd := exec.Command("ssh-keygen", "-lf", "-")
	cmd.Stdin = strings.NewReader(line)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err == nil {
		// Parse output: "4096 SHA256:abcdef... comment (RSA)\n"
		fpFields := strings.Fields(stdout.String())
		if len(fpFields) >= 2 {
			key.Fingerprint = fpFields[1]
			if len(fpFields) >= 1 {
				if size, err := strconv.Atoi(fpFields[0]); err == nil {
					key.Size = size
				}
			}
		}
	}

	return key
}

// mergeAgentKeysWithFilesystem merges agent keys with filesystem keys to populate Path field.
func (gs *gitService) mergeAgentKeysWithFilesystem(keys []domain.SSHKey) []domain.SSHKey {
	fsKeys, err := gs.ListSSHKeysFromConfig(gs.serverRepository)
	if err != nil {
		return keys
	}

	gs.logger.Debugf("mergeAgentKeysWithFilesystem: %d agent keys, %d filesystem keys", len(keys), len(fsKeys))

	// Create maps for different matching strategies
	fsKeyByPublicKeyLine := make(map[string]domain.SSHKey)
	fsKeyByFingerprint := make(map[string]domain.SSHKey)
	fsKeyByName := make(map[string]domain.SSHKey)

	for _, fsKey := range fsKeys {
		if fsKey.PublicKeyLine != "" {
			fsKeyByPublicKeyLine[fsKey.PublicKeyLine] = fsKey
		}
		if fsKey.Fingerprint != "" {
			fsKeyByFingerprint[fsKey.Fingerprint] = fsKey
			gs.logger.Debugf("  FS key fingerprint map: %s -> Path=%s", fsKey.Fingerprint, fsKey.Path)
		}
		if fsKey.Name != "" {
			fsKeyByName[fsKey.Name] = fsKey
		}
	}

	// Update agent keys with path from filesystem
	for i := range keys {
		gs.logger.Debugf("Processing agent key [%d]: Name=%q, FP=%q, Path=%q",
			i, keys[i].Name, keys[i].Fingerprint, keys[i].Path)
		fsKey, matched := gs.matchAgentKeyToFilesystem(&keys[i], fsKeyByPublicKeyLine, fsKeyByFingerprint, fsKeyByName)

		if !matched {
			gs.logger.Debugf("  -> No match found, Path remains empty")
			continue
		}

		// Populate the Path field and copy other attributes
		keys[i].Path = fsKey.Path
		// Always update comment from filesystem (it's the source of truth after editing)
		if fsKey.Comment != "" {
			keys[i].Comment = fsKey.Comment
		}
		if fsKey.Name != "" && len(fsKey.Name) > len(keys[i].Name) {
			keys[i].Name = fsKey.Name
		}
		if keys[i].Fingerprint == "" && fsKey.Fingerprint != "" {
			keys[i].Fingerprint = fsKey.Fingerprint
		}
		if keys[i].Type == "" && fsKey.Type != "" {
			keys[i].Type = fsKey.Type
		}
		if keys[i].Size == 0 && fsKey.Size > 0 {
			keys[i].Size = fsKey.Size
		}
		// Copy HasPublicKey from filesystem key
		if fsKey.HasPublicKey {
			keys[i].HasPublicKey = true
		}
		// Copy IsEncrypted from filesystem key
		if fsKey.IsEncrypted {
			keys[i].IsEncrypted = true
		}
	}

	return keys
}

// matchAgentKeyToFilesystem tries to match an agent key to a filesystem key using multiple strategies.
func (gs *gitService) matchAgentKeyToFilesystem(
	key *domain.SSHKey,
	fsKeyByPublicKeyLine, fsKeyByFingerprint, fsKeyByName map[string]domain.SSHKey,
) (*domain.SSHKey, bool) {
	// Strategy 1: Match by public key line (most reliable)
	if key.PublicKeyLine != "" {
		if k, ok := fsKeyByPublicKeyLine[key.PublicKeyLine]; ok {
			gs.logger.Debugf("  -> Matched by public key line to FS key: Path=%s", k.Path)
			return &k, true
		}
	}

	// Strategy 2: Match by fingerprint
	if key.Fingerprint != "" {
		gs.logger.Debugf("  -> Trying to match by fingerprint: agent FP=%s", key.Fingerprint)
		if k, ok := fsKeyByFingerprint[key.Fingerprint]; ok {
			gs.logger.Debugf("  -> Matched by fingerprint to FS key: Path=%s", k.Path)
			return &k, true
		} else {
			gs.logger.Debugf("  -> No fingerprint match found in filesystem keys")
		}
	}

	// Strategy 3: Match by filename (fallback)
	if key.Name != "" {
		if k, ok := fsKeyByName[key.Name]; ok {
			gs.logger.Debugf("  -> Matched by filename to FS key: Path=%s", k.Path)
			return &k, true
		}
	}

	return nil, false
}

// LoadKeyToAgent loads an SSH key into ssh-agent.
func (gs *gitService) LoadKeyToAgent(keyPath string) error {
	cmd := exec.Command("ssh-add", keyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// UnloadKeyFromAgent removes an SSH key from ssh-agent using its public key line.
// The publicKeyLine should be the full line from ssh-add -L output (e.g., "ssh-rsa AAAAB3... comment")
func (gs *gitService) UnloadKeyFromAgent(publicKeyLine string) error {
	if publicKeyLine == "" {
		return fmt.Errorf("empty public key line provided")
	}

	// Capture stderr for better error messages
	var stderrBuf bytes.Buffer
	cmd := exec.Command("ssh-add", "-d", "-")
	cmd.Stdin = strings.NewReader(publicKeyLine + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrMsg := stderrBuf.String()
		if stderrMsg != "" {
			return fmt.Errorf("failed to unload key: %w (ssh-add output: %s)", err, stderrMsg)
		}
		return fmt.Errorf("failed to unload key: %w", err)
	}

	gs.logger.Infof("Successfully unloaded key")
	return nil
}

// UpdateKeyComment updates the comment for an SSH key using ssh-keygen.
// Handles file permissions by temporarily making the key writable if needed.
func (gs *gitService) UpdateKeyComment(keyPath, comment string) error {
	// Check if private key file exists
	if _, err := os.Stat(keyPath); err != nil {
		return fmt.Errorf("key file not found: %w", err)
	}

	// Check if public key file exists (required for ssh-keygen -c)
	pubKeyPath := keyPath + ".pub"
	if _, err := os.Stat(pubKeyPath); err != nil {
		return fmt.Errorf("public key file (.pub) not found: %w", err)
	}

	// Get original file permissions for both files
	originalMode, err := getFileMode(keyPath)
	if err != nil {
		return fmt.Errorf("failed to get private key file permissions: %w", err)
	}
	pubOriginalMode, err := getFileMode(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to get public key file permissions: %w", err)
	}

	// Check if files are writable
	privateReadonly := originalMode&0o200 == 0
	pubReadonly := pubOriginalMode&0o200 == 0

	// Make private key writable if it's readonly
	if privateReadonly {
		if err := os.Chmod(keyPath, 0o600); err != nil {
			return fmt.Errorf("failed to make private key file writable: %w", err)
		}
	}

	// Make public key writable if it's readonly
	if pubReadonly {
		if err := os.Chmod(pubKeyPath, 0o600); err != nil {
			// Restore private key permissions on error
			if privateReadonly {
				_ = os.Chmod(keyPath, originalMode)
			}
			return fmt.Errorf("failed to make public key file writable: %w", err)
		}
	}

	// Use ssh-keygen to update the comment
	// #nosec G204 -- keyPath and comment are validated inputs
	cmd := exec.Command("ssh-keygen", "-c", "-C", comment, "-f", keyPath)
	// Connect stdin/stdout/stderr for interactive passphrase prompt (for encrypted keys)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Restore original permissions on error
		if privateReadonly {
			_ = os.Chmod(keyPath, originalMode)
		}
		if pubReadonly {
			_ = os.Chmod(pubKeyPath, pubOriginalMode)
		}
		return fmt.Errorf("failed to update key comment: %w", err)
	}

	// Restore original permissions if they were readonly
	if privateReadonly {
		if err := os.Chmod(keyPath, originalMode); err != nil {
			gs.logger.Warnw("failed to restore original private key file permissions", "path", keyPath, "error", err)
			// Don't return error as the comment update succeeded
		}
	}
	if pubReadonly {
		if err := os.Chmod(pubKeyPath, pubOriginalMode); err != nil {
			gs.logger.Warnw("failed to restore original public key file permissions", "path", pubKeyPath, "error", err)
			// Don't return error as the comment update succeeded
		}
	}

	gs.logger.Infof("Successfully updated key comment for %s", keyPath)
	return nil
}

// getFileMode gets the file mode, using os.Stat if Lstat fails (for symlinks)
func getFileMode(path string) (os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		// Try Lstat if Stat fails
		info, err = os.Lstat(path)
		if err != nil {
			return 0, err
		}
	}
	return info.Mode(), nil
}
