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
	"strconv"
	"strings"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

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

// mapKVToServer maps an ssh_config.KV node to the corresponding fields in domain.Server.
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
