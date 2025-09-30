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

import "testing"

func TestGenerateUniqueAlias(t *testing.T) {
	tests := []struct {
		name            string
		baseAlias       string
		existingAliases []string
		want            string
	}{
		{
			name:            "no duplicates",
			baseAlias:       "server",
			existingAliases: []string{"other", "another"},
			want:            "server",
		},
		{
			name:            "one duplicate",
			baseAlias:       "server",
			existingAliases: []string{"server", "other"},
			want:            "server_1",
		},
		{
			name:            "multiple duplicates in sequence",
			baseAlias:       "server",
			existingAliases: []string{"server", "server_1", "server_2"},
			want:            "server_3",
		},
		{
			name:            "duplicates with gap",
			baseAlias:       "server",
			existingAliases: []string{"server", "server_1", "server_5"},
			want:            "server_6",
		},
		{
			name:            "empty existing aliases",
			baseAlias:       "server",
			existingAliases: []string{},
			want:            "server",
		},
		{
			name:            "nil existing aliases",
			baseAlias:       "server",
			existingAliases: nil,
			want:            "server",
		},
		{
			name:            "copying alias with suffix",
			baseAlias:       "123_1",
			existingAliases: []string{"123", "123_1"},
			want:            "123_2",
		},
		{
			name:            "copying alias with suffix and gap",
			baseAlias:       "server_2",
			existingAliases: []string{"server", "server_1", "server_2", "server_5"},
			want:            "server_6",
		},
		{
			name:            "alias with underscore but not numeric suffix",
			baseAlias:       "my_server",
			existingAliases: []string{"my_server"},
			want:            "my_server_1",
		},
		{
			name:            "complex case with numeric alias",
			baseAlias:       "123",
			existingAliases: []string{"123"},
			want:            "123_1",
		},
		{
			name:            "copy already suffixed numeric alias",
			baseAlias:       "123_1",
			existingAliases: []string{"123", "123_1", "123_2"},
			want:            "123_3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateUniqueAlias(tt.baseAlias, tt.existingAliases)
			if got != tt.want {
				t.Errorf("GenerateUniqueAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateSmartAlias(t *testing.T) {
	tests := []struct {
		name string
		host string
		user string
		port int
		want string
	}{
		{
			name: "simple domain",
			host: "example.com",
			user: "",
			port: 22,
			want: "example",
		},
		{
			name: "www prefix",
			host: "www.example.com",
			user: "",
			port: 22,
			want: "example",
		},
		{
			name: "subdomain",
			host: "api.github.com",
			user: "",
			port: 22,
			want: "api.github",
		},
		{
			name: "complex subdomain",
			host: "dev.server.example.com",
			user: "",
			port: 22,
			want: "dev.server",
		},
		{
			name: "IP address",
			host: "192.168.1.1",
			user: "",
			port: 22,
			want: "192.168.1.1",
		},
		{
			name: "with non-standard port",
			host: "example.com",
			user: "",
			port: 2222,
			want: "example:2222",
		},
		{
			name: "with custom user",
			host: "example.com",
			user: "developer",
			port: 22,
			want: "developer@example",
		},
		{
			name: "with common user (root)",
			host: "example.com",
			user: "root",
			port: 22,
			want: "example",
		},
		{
			name: "with common user (ubuntu)",
			host: "example.com",
			user: "ubuntu",
			port: 22,
			want: "example",
		},
		{
			name: "with common user (ec2-user)",
			host: "example.com",
			user: "ec2-user",
			port: 22,
			want: "example",
		},
		{
			name: "with common user (centos)",
			host: "example.com",
			user: "centos",
			port: 22,
			want: "example",
		},
		{
			name: "with common user (azureuser)",
			host: "example.com",
			user: "azureuser",
			port: 22,
			want: "example",
		},
		{
			name: "full combination",
			host: "api.github.com",
			user: "developer",
			port: 2222,
			want: "developer@api.github:2222",
		},
		{
			name: "IPv6 address",
			host: "2001:db8::1",
			user: "",
			port: 22,
			want: "2001:db8::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateSmartAlias(tt.host, tt.user, tt.port)
			if got != tt.want {
				t.Errorf("GenerateSmartAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}
