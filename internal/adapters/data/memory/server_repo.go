package memory

import (
	"github.com/Adembc/lazyssh/internal/core/domain"
	"strconv"
	"strings"
	"time"
)

type serverRepository struct {
}

var servers = []domain.Server{
	{Alias: "web-01", Host: "192.168.1.10", User: "root", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "web"}, Status: "online", LastSeen: time.Now().Add(-2 * time.Hour)},
	{Alias: "web-02", Host: "192.168.1.11", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "web"}, Status: "warn", LastSeen: time.Now().Add(-30 * time.Minute)},
	{Alias: "db-01", Host: "192.168.1.20", User: "postgres", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "db"}, Status: "offline", LastSeen: time.Now().Add(-26 * time.Hour)},
	{Alias: "api-01", Host: "192.168.1.30", User: "deploy", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "api"}, Status: "online", LastSeen: time.Now().Add(-10 * time.Minute)},
	{Alias: "cache-01", Host: "192.168.1.40", User: "redis", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "cache"}, Status: "online", LastSeen: time.Now().Add(-1 * time.Hour)},
	{Alias: "dev-web", Host: "10.0.1.10", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "web"}, Status: "online", LastSeen: time.Now().Add(-5 * time.Minute)},
	{Alias: "dev-db", Host: "10.0.1.20", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "db"}, Status: "online", LastSeen: time.Now().Add(-15 * time.Minute)},
	{Alias: "staging", Host: "staging.example.com", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"test"}, Status: "warn", LastSeen: time.Now().Add(-45 * time.Minute)},
}

// NewServerRepository creates a new server repository with the given file path.
func NewServerRepository() *serverRepository {
	return &serverRepository{}
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
		status := strings.ToLower(server.Status)
		port := strconv.Itoa(server.Port)

		match := false
		if strings.Contains(alias, q) || strings.Contains(host, q) || strings.Contains(user, q) || strings.Contains(status, q) || strings.Contains(port, q) {
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
	return servers, nil
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
