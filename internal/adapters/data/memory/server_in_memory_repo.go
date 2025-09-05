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

package memory

import (
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/kjm0001/lazyssh/internal/core/domain"
)

type serverRepository struct {
	logger *zap.SugaredLogger
}

var servers = []domain.Server{
	{Alias: "web-01", Host: "192.168.1.10", User: "root", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "web"}, LastSeen: time.Now().Add(-2 * time.Hour)},
	{Alias: "web-02", Host: "192.168.1.11", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "web"}, LastSeen: time.Now().Add(-30 * time.Minute)},
	{Alias: "db-01", Host: "192.168.1.20", User: "postgres", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "db"}, LastSeen: time.Now().Add(-26 * time.Hour)},
	{Alias: "api-01", Host: "192.168.1.30", User: "deploy", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "api"}, LastSeen: time.Now().Add(-10 * time.Minute)},
	{Alias: "cache-01", Host: "192.168.1.40", User: "redis", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "cache"}, LastSeen: time.Now().Add(-1 * time.Hour)},
	{Alias: "dev-web", Host: "10.0.1.10", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "web"}, LastSeen: time.Now().Add(-5 * time.Minute)},
	{Alias: "dev-db", Host: "10.0.1.20", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "db"}, LastSeen: time.Now().Add(-15 * time.Minute)},
	{Alias: "staging", Host: "staging.example.com", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"test"}, LastSeen: time.Now().Add(-45 * time.Minute)},
}

// NewServerRepository creates a new server repository with the given file path.
func NewServerRepository(logger *zap.SugaredLogger) *serverRepository {
	return &serverRepository{
		logger: logger,
	}
}

// ListServers returns a list of servers from the repository.
func (r *serverRepository) ListServers(query string) ([]domain.Server, error) {
	if query == "" {
		return servers, nil
	}
	q := strings.ToLower(strings.TrimSpace(query))

	var filteredServers []domain.Server
	for _, server := range servers {
		alias := strings.ToLower(server.Alias)
		host := strings.ToLower(server.Host)
		user := strings.ToLower(server.User)
		port := strconv.Itoa(server.Port)

		match := false
		if strings.Contains(alias, q) || strings.Contains(host, q) || strings.Contains(user, q) || strings.Contains(port, q) {
			match = true
		}
		if !match {
			for _, tag := range server.Tags {
				if strings.Contains(strings.ToLower(tag), q) {
					match = true
					break
				}
			}
		}
		if match {
			filteredServers = append(filteredServers, server)
		}
	}
	return filteredServers, nil
}

// UpdateServer updates an existing server with new details.
func (r *serverRepository) UpdateServer(server domain.Server, newServer domain.Server) error {
	for i, s := range servers {
		if s.Alias == server.Alias {
			servers[i] = newServer
			return nil
		}
	}
	return nil
}

// AddServer adds a new server to the repository.
func (r *serverRepository) AddServer(server domain.Server) error {
	servers = append(servers, server)
	return nil
}

// DeleteServer removes a server from the repository.
func (r *serverRepository) DeleteServer(server domain.Server) error {
	for i, s := range servers {
		if s.Alias == server.Alias {
			servers = append(servers[:i], servers[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *serverRepository) SetPinned(alias string, pinned bool) error {
	for i, s := range servers {
		if s.Alias == alias {
			if pinned {
				servers[i].PinnedAt = time.Now()
			} else {
				servers[i].PinnedAt = time.Time{}
			}
			return nil
		}
	}
	return nil
}

func (r *serverRepository) RecordSSH(alias string) error {
	for i, s := range servers {
		if s.Alias == alias {
			servers[i].LastSeen = time.Now()
			return nil
		}
	}
	return nil
}
