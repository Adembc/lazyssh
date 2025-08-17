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
