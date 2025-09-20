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
	"strings"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

func TestBuildSSHCommand_PortForwarding(t *testing.T) {
	tests := []struct {
		name     string
		server   domain.Server
		expected []string // expected parts in the command
	}{
		{
			name: "local forward",
			server: domain.Server{
				Alias:        "test",
				Host:         "example.com",
				User:         "user",
				LocalForward: []string{"8080:localhost:80", "3306:db.internal:3306"},
			},
			expected: []string{"ssh", "-L", "8080:localhost:80", "-L", "3306:db.internal:3306", "user@example.com"},
		},
		{
			name: "remote forward",
			server: domain.Server{
				Alias:         "test",
				Host:          "example.com",
				User:          "user",
				RemoteForward: []string{"8080:localhost:3000", "*:80:localhost:8080"},
			},
			expected: []string{"ssh", "-R", "8080:localhost:3000", "-R", "*:80:localhost:8080", "user@example.com"},
		},
		{
			name: "dynamic forward",
			server: domain.Server{
				Alias:          "test",
				Host:           "example.com",
				User:           "user",
				DynamicForward: []string{"1080", "localhost:1081"},
			},
			expected: []string{"ssh", "-D", "1080", "-D", "localhost:1081", "user@example.com"},
		},
		{
			name: "all forward types",
			server: domain.Server{
				Alias:          "test",
				Host:           "example.com",
				User:           "user",
				LocalForward:   []string{"8080:localhost:80"},
				RemoteForward:  []string{"9090:localhost:9090"},
				DynamicForward: []string{"1080"},
			},
			expected: []string{"ssh", "-L", "8080:localhost:80", "-R", "9090:localhost:9090", "-D", "1080", "user@example.com"},
		},
		{
			name: "forward with bind address",
			server: domain.Server{
				Alias:        "test",
				Host:         "example.com",
				User:         "user",
				LocalForward: []string{"127.0.0.1:8080:localhost:80", "*:3000:localhost:3000"},
			},
			expected: []string{"ssh", "-L", "127.0.0.1:8080:localhost:80", "-L", "*:3000:localhost:3000", "user@example.com"},
		},
		{
			name: "forward with additional options",
			server: domain.Server{
				Alias:                "test",
				Host:                 "example.com",
				User:                 "user",
				LocalForward:         []string{"8080:localhost:80"},
				ExitOnForwardFailure: "yes",
				GatewayPorts:         "clientspecified",
			},
			expected: []string{"ssh", "-L", "8080:localhost:80", "-o", "ExitOnForwardFailure=yes", "-o", "GatewayPorts=clientspecified", "user@example.com"},
		},
		{
			name: "clear all forwardings",
			server: domain.Server{
				Alias:               "test",
				Host:                "example.com",
				User:                "user",
				LocalForward:        []string{"8080:localhost:80"},
				ClearAllForwardings: "yes",
			},
			expected: []string{"ssh", "-L", "8080:localhost:80", "-o", "ClearAllForwardings=yes", "user@example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSSHCommand(tt.server)

			// Check that all expected parts are in the result
			for _, part := range tt.expected {
				if !strings.Contains(result, part) {
					t.Errorf("BuildSSHCommand() missing expected part %q in result: %q", part, result)
				}
			}

			// Additional check: ensure the command starts with "ssh"
			if !strings.HasPrefix(result, "ssh ") {
				t.Errorf("BuildSSHCommand() should start with 'ssh ', got: %q", result)
			}
		})
	}
}

func TestBuildSSHCommand_CompleteCommand(t *testing.T) {
	server := domain.Server{
		Alias:          "myserver",
		Host:           "example.com",
		User:           "admin",
		Port:           2222,
		LocalForward:   []string{"8080:localhost:80", "3306:db.internal:3306"},
		RemoteForward:  []string{"9090:localhost:9090"},
		DynamicForward: []string{"1080"},
		IdentityFiles:  []string{"~/.ssh/id_rsa"},
	}

	result := BuildSSHCommand(server)

	// Check command structure
	if !strings.HasPrefix(result, "ssh ") {
		t.Errorf("Command should start with 'ssh ', got: %q", result)
	}

	// Check port
	if !strings.Contains(result, "-p 2222") {
		t.Errorf("Command should contain port flag '-p 2222', got: %q", result)
	}

	// Check identity file
	if !strings.Contains(result, "-i ~/.ssh/id_rsa") {
		t.Errorf("Command should contain identity file flag, got: %q", result)
	}

	// Check all forwards
	expectedForwards := []string{
		"-L 8080:localhost:80",
		"-L 3306:db.internal:3306",
		"-R 9090:localhost:9090",
		"-D 1080",
	}

	for _, forward := range expectedForwards {
		if !strings.Contains(result, forward) {
			t.Errorf("Command should contain forward %q, got: %q", forward, result)
		}
	}

	// Check user@host
	if !strings.Contains(result, "admin@example.com") {
		t.Errorf("Command should contain 'admin@example.com', got: %q", result)
	}
}

func TestFilterServersByHidden(t *testing.T) {
	servers := []domain.Server{
		{Alias: "a", Hidden: false},
		{Alias: "b", Hidden: true},
		{Alias: "c", Hidden: true},
		{Alias: "d", Hidden: false},
	}

	filtered, hiddenCount := filterServersByHidden(servers, false)
	if hiddenCount != 2 {
		t.Fatalf("expected hiddenCount=2, got %d", hiddenCount)
	}
	if len(filtered) != 2 {
		t.Fatalf("expected 2 visible servers, got %d", len(filtered))
	}
	for _, srv := range filtered {
		if srv.Hidden {
			t.Fatalf("expected filtered servers to exclude hidden ones, got %+v", srv)
		}
	}

	filtered, hiddenCount = filterServersByHidden(servers, true)
	if hiddenCount != 2 {
		t.Fatalf("expected hiddenCount to remain 2 when including hidden, got %d", hiddenCount)
	}
	if len(filtered) != len(servers) {
		t.Fatalf("expected all servers when including hidden, got %d", len(filtered))
	}
}
