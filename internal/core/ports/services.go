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

package ports

import (
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

type ServerService interface {
	ListServers(query string) ([]domain.Server, error)
	UpdateServer(server domain.Server, newServer domain.Server) error
	AddServer(server domain.Server) error
	DeleteServer(server domain.Server) error
	SetPinned(alias string, pinned bool) error
	SSH(alias string) error
	SSHWithArgs(alias string, extraArgs []string) error
	StartForward(alias string, extraArgs []string) (int, error)
	StopForwarding(alias string) error
	IsForwarding(alias string) bool
	Ping(server domain.Server) (bool, time.Duration, error)
}

type SSHKey struct {
	Name        string
	Path        string
	HasPubKey   bool
	ModTime     time.Time
	IsEncrypted bool
	InAgent     bool // Whether the key is loaded in ssh-agent/keychain
}

type GitService interface {
	IsGitRepository(path string) bool
	GetGitRootPath(path string) (string, error)
	GetPushRemoteURL(repoPath string) (remoteName, remoteURL string, err error)
	ListSSHKeys(sshDir string, serverRepo ServerRepository) ([]SSHKey, error)
	ConfigureGitSSHKey(repoPath string, keyPath string, scope string) error
	GetCurrentGitSSHConfig(repoPath string) (string, error)
	ClearGitSSHConfig(repoPath string, scope string) error
	GetLoadedAgentKeys() ([]string, error)

	// SSH Key management
	ListAllSSHKeys(serverRepo ServerRepository) ([]domain.SSHKey, error)
	ListSSHKeysFromConfig(serverRepo ServerRepository) ([]domain.SSHKey, error)
	ListSSHKeysFromAgent() ([]domain.SSHKey, error)
	LoadKeyToAgent(keyPath string) error
	UnloadKeyFromAgent(publicKeyLine string) error
	UpdateKeyComment(keyPath, comment string) error

	// SetServerRepository sets the server repository for SSH key path resolution
	SetServerRepository(repo ServerRepository)
}
