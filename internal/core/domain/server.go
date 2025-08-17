package domain

import "time"

type Server struct {
	Alias    string
	Host     string
	User     string
	Port     int
	Key      string
	Tags     []string
	Status   string
	LastSeen time.Time
}
