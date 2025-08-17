package services

import (
	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
)

type serverService struct {
	serverRepository ports.ServerRepository
}

// NewServerService creates a new instance of serverService.
func NewServerService(sr ports.ServerRepository) *serverService {
	return &serverService{
		serverRepository: sr,
	}
}

// ListServers returns a list of servers.
func (s *serverService) ListServers(query string) ([]domain.Server, error) {
	// do any relevant business logic here if needed
	servers, err := s.serverRepository.ListServers(query)
	if err != nil {
		return nil, err
	}
	return servers, nil
}

// UpdateServer updates an existing server with new details.
func (s *serverService) UpdateServer(server domain.Server, newServer domain.Server) error {
	return s.serverRepository.UpdateServer(server, newServer)
}

// AddServer adds a new server to the repository.
func (s *serverService) AddServer(server domain.Server) error {
	return s.serverRepository.AddServer(server)
}

// DeleteServer removes a server from the repository.
func (s *serverService) DeleteServer(server domain.Server) error {
	return s.serverRepository.DeleteServer(server)
}
