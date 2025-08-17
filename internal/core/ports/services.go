package ports

import "github.com/Adembc/lazyssh/internal/core/domain"

type ServerService interface {
	ListServers(query string) ([]domain.Server, error)
	UpdateServer(server domain.Server, newServer domain.Server) error
	AddServer(server domain.Server) error
	DeleteServer(server domain.Server) error
}
