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

func TestBuildSSHCommand_AliasAndTags(t *testing.T) {
	tests := []struct {
		name         string
		server       domain.Server
		wantContains []string
		wantPrefix   string
	}{
		{
			name: "with alias only",
			server: domain.Server{
				Alias: "myserver",
				Host:  "example.com",
				User:  "user",
			},
			wantPrefix: "# lazyssh-alias:myserver",
			wantContains: []string{
				"# lazyssh-alias:myserver",
				"\nssh ",
				"user@example.com",
			},
		},
		{
			name: "with alias and tags",
			server: domain.Server{
				Alias: "prod-server",
				Host:  "prod.example.com",
				User:  "admin",
				Tags:  []string{"production", "critical", "web"},
			},
			wantPrefix: "# lazyssh-alias:prod-server tags:production,critical,web",
			wantContains: []string{
				"# lazyssh-alias:prod-server tags:production,critical,web",
				"\nssh ",
				"admin@prod.example.com",
			},
		},
		{
			name: "with alias and single tag",
			server: domain.Server{
				Alias: "dev-server",
				Host:  "dev.example.com",
				User:  "developer",
				Tags:  []string{"development"},
			},
			wantPrefix: "# lazyssh-alias:dev-server tags:development",
			wantContains: []string{
				"# lazyssh-alias:dev-server tags:development",
				"\nssh ",
				"developer@dev.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSSHCommand(tt.server)

			// Check prefix
			if !strings.HasPrefix(result, tt.wantPrefix) {
				t.Errorf("BuildSSHCommand() prefix = %q, want prefix %q", result[:len(tt.wantPrefix)], tt.wantPrefix)
			}

			// Check contains
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("BuildSSHCommand() missing %q in result: %q", want, result)
				}
			}
		})
	}
}
