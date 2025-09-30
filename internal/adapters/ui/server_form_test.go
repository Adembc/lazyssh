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

package ui

import (
	"testing"
	"time"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

func TestServersDifferIgnoresTransientFields(t *testing.T) {
	sf := &ServerForm{}

	// Create two servers with identical config but different transient fields
	server1 := domain.Server{
		Alias:       "test-server",
		Host:        "example.com",
		User:        "testuser",
		Port:        22,
		PingStatus:  "up",
		PingLatency: 100 * time.Millisecond,
		LastSeen:    time.Now(),
		PinnedAt:    time.Now(),
		SSHCount:    5,
	}

	server2 := domain.Server{
		Alias:       "test-server",
		Host:        "example.com",
		User:        "testuser",
		Port:        22,
		PingStatus:  "down",                        // Different transient field
		PingLatency: 200 * time.Millisecond,        // Different transient field
		LastSeen:    time.Now().Add(1 * time.Hour), // Different metadata field
		PinnedAt:    time.Now().Add(2 * time.Hour), // Different metadata field
		SSHCount:    10,                            // Different metadata field
	}

	// Should not detect differences since only transient/metadata fields differ
	if sf.serversDiffer(server1, server2) {
		t.Error("serversDiffer should ignore transient and metadata fields")
	}

	// Now change a real config field
	server2.Port = 2222

	// Should detect the difference now
	if !sf.serversDiffer(server1, server2) {
		t.Error("serversDiffer should detect differences in non-transient fields")
	}
}

func TestServersDifferDetectsRealChanges(t *testing.T) {
	sf := &ServerForm{}

	server1 := domain.Server{
		Alias: "test-server",
		Host:  "example.com",
		User:  "testuser",
		Port:  22,
	}

	testCases := []struct {
		name   string
		modify func(*domain.Server)
		expect bool
	}{
		{
			name:   "No changes",
			modify: func(s *domain.Server) {},
			expect: false,
		},
		{
			name:   "Changed Host",
			modify: func(s *domain.Server) { s.Host = "different.com" },
			expect: true,
		},
		{
			name:   "Changed User",
			modify: func(s *domain.Server) { s.User = "otheruser" },
			expect: true,
		},
		{
			name:   "Changed Port",
			modify: func(s *domain.Server) { s.Port = 2222 },
			expect: true,
		},
		{
			name:   "Added IdentityFile",
			modify: func(s *domain.Server) { s.IdentityFiles = []string{"~/.ssh/id_rsa"} },
			expect: true,
		},
		{
			name:   "Changed ProxyJump",
			modify: func(s *domain.Server) { s.ProxyJump = "jumphost" },
			expect: true,
		},
		{
			name:   "Changed only PingStatus (transient)",
			modify: func(s *domain.Server) { s.PingStatus = "checking" },
			expect: false,
		},
		{
			name:   "Changed only LastSeen (metadata)",
			modify: func(s *domain.Server) { s.LastSeen = time.Now().Add(1 * time.Hour) },
			expect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server2 := server1 // Copy
			tc.modify(&server2)
			result := sf.serversDiffer(server1, server2)
			if result != tc.expect {
				t.Errorf("Expected %v but got %v for test case %s", tc.expect, result, tc.name)
			}
		})
	}
}
