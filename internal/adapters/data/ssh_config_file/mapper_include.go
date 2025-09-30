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
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

// toDomainServersFromConfig converts ssh_config.Config to a slice of domain.Server,
// setting origin metadata (SourceFile, Readonly).
func (r *Repository) toDomainServersFromConfig(cfg *ssh_config.Config, origin string, isMain bool) []domain.Server {
	servers := make([]domain.Server, 0, len(cfg.Hosts))
	for _, host := range cfg.Hosts {

		aliases := make([]string, 0, len(host.Patterns))
		for _, pattern := range host.Patterns {
			alias := pattern.String()
			// Skip patterns with wildcards
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

			SourceFile: origin,
			Readonly:   !isMain,
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
