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

	"github.com/Adembc/lazyssh/internal/core/domain"
)

const (
	testHost = "example.com"
	testUser = "user"
)

func TestParseSSHCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
		check   func(t *testing.T, server interface{})
	}{
		{
			name: "basic ssh command",
			cmd:  "ssh user@" + testHost,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.User != testUser {
					t.Errorf("expected user '%s', got %s", testUser, server.User)
				}
				if server.Host != testHost {
					t.Errorf("expected host '%s', got %s", testHost, server.Host)
				}
			},
		},
		{
			name: "ssh with port",
			cmd:  "ssh -p 2222 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
			},
		},
		{
			name: "ssh with port after host",
			cmd:  "ssh user@prod-server-01.example.com -p 2222",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
				if server.Host != "prod-server-01.example.com" {
					t.Errorf("expected host prod-server-01.example.com, got %s", server.Host)
				}
				if server.User != "user" {
					t.Errorf("expected user 'user', got %s", server.User)
				}
			},
		},
		{
			name: "ssh with port after host and remote command",
			cmd:  "ssh user@example.com -p 2222 ls -la",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
				if server.RemoteCommand != "ls -la" {
					t.Errorf("expected remote command 'ls -la', got %s", server.RemoteCommand)
				}
			},
		},
		{
			name: "ssh with mixed flags before and after host",
			cmd:  "ssh -i ~/.ssh/id_rsa user@example.com -p 2222 -v",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
				if len(server.IdentityFiles) != 1 || server.IdentityFiles[0] != "~/.ssh/id_rsa" {
					t.Errorf("expected identity file '~/.ssh/id_rsa', got %v", server.IdentityFiles)
				}
				if server.LogLevel != "VERBOSE" {
					t.Errorf("expected log level 'VERBOSE', got %s", server.LogLevel)
				}
			},
		},
		{
			name: "ssh with identity file",
			cmd:  "ssh -i ~/.ssh/id_rsa user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.IdentityFiles) != 1 || server.IdentityFiles[0] != "~/.ssh/id_rsa" {
					t.Errorf("expected identity file '~/.ssh/id_rsa', got %v", server.IdentityFiles)
				}
			},
		},
		{
			name: "ssh with multiple identity files",
			cmd:  "ssh -i ~/.ssh/id_rsa -i ~/.ssh/id_ed25519 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.IdentityFiles) != 2 {
					t.Errorf("expected 2 identity files, got %d", len(server.IdentityFiles))
				}
			},
		},
		{
			name: "ssh with proxy jump",
			cmd:  "ssh -J bastion@jump.example.com user@internal.example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.ProxyJump != "bastion@jump.example.com" {
					t.Errorf("expected ProxyJump 'bastion@jump.example.com', got %s", server.ProxyJump)
				}
			},
		},
		{
			name: "ssh with local forwarding",
			cmd:  "ssh -L 8080:localhost:80 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.LocalForward) != 1 || server.LocalForward[0] != "8080:localhost:80" {
					t.Errorf("expected local forward '8080:localhost:80', got %v", server.LocalForward)
				}
			},
		},
		{
			name: "ssh with remote forwarding",
			cmd:  "ssh -R 9090:localhost:9090 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.RemoteForward) != 1 || server.RemoteForward[0] != "9090:localhost:9090" {
					t.Errorf("expected remote forward '9090:localhost:9090', got %v", server.RemoteForward)
				}
			},
		},
		{
			name: "ssh with dynamic forwarding",
			cmd:  "ssh -D 1080 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.DynamicForward) != 1 || server.DynamicForward[0] != "1080" {
					t.Errorf("expected dynamic forward '1080', got %v", server.DynamicForward)
				}
			},
		},
		{
			name: "ssh with agent forwarding",
			cmd:  "ssh -A user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.ForwardAgent != sshYes {
					t.Errorf("expected ForwardAgent 'yes', got %s", server.ForwardAgent)
				}
			},
		},
		{
			name: "ssh with X11 forwarding",
			cmd:  "ssh -X user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.ForwardX11 != sshYes {
					t.Errorf("expected ForwardX11 'yes', got %s", server.ForwardX11)
				}
			},
		},
		{
			name: "ssh with trusted X11 forwarding",
			cmd:  "ssh -Y user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.ForwardX11 != sshYes {
					t.Errorf("expected ForwardX11 'yes', got %s", server.ForwardX11)
				}
				if server.ForwardX11Trusted != sshYes {
					t.Errorf("expected ForwardX11Trusted 'yes', got %s", server.ForwardX11Trusted)
				}
			},
		},
		{
			name: "ssh with compression",
			cmd:  "ssh -C user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Compression != sshYes {
					t.Errorf("expected Compression 'yes', got %s", server.Compression)
				}
			},
		},
		{
			name: "ssh with -o options",
			cmd:  "ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.StrictHostKeyChecking != sshNo {
					t.Errorf("expected StrictHostKeyChecking 'no', got %s", server.StrictHostKeyChecking)
				}
				if server.ConnectTimeout != "10" {
					t.Errorf("expected ConnectTimeout '10', got %s", server.ConnectTimeout)
				}
			},
		},
		{
			name: "ssh with complex forwarding",
			cmd:  "ssh -L 8080:localhost:80 -L 3306:db:3306 -R 9090:localhost:9090 -D 1080 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if len(server.LocalForward) != 2 {
					t.Errorf("expected 2 local forwards, got %d", len(server.LocalForward))
				}
				if len(server.RemoteForward) != 1 {
					t.Errorf("expected 1 remote forward, got %d", len(server.RemoteForward))
				}
				if len(server.DynamicForward) != 1 {
					t.Errorf("expected 1 dynamic forward, got %d", len(server.DynamicForward))
				}
			},
		},
		{
			name: "ssh with remote command",
			cmd:  `ssh user@example.com "ls -la /tmp"`,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.RemoteCommand != "ls -la /tmp" {
					t.Errorf("expected RemoteCommand 'ls -la /tmp', got %s", server.RemoteCommand)
				}
			},
		},
		{
			name: "ssh with login name flag",
			cmd:  "ssh -l admin example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.User != "admin" {
					t.Errorf("expected user 'admin', got %s", server.User)
				}
			},
		},
		{
			name: "ssh with IPv4 only",
			cmd:  "ssh -4 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.AddressFamily != "inet" {
					t.Errorf("expected AddressFamily 'inet', got %s", server.AddressFamily)
				}
			},
		},
		{
			name: "ssh with IPv6 only",
			cmd:  "ssh -6 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.AddressFamily != "inet6" {
					t.Errorf("expected AddressFamily 'inet6', got %s", server.AddressFamily)
				}
			},
		},
		{
			name: "ssh with control master",
			cmd:  "ssh -M -S /tmp/ssh_control_%h_%p user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.ControlMaster != sshYes {
					t.Errorf("expected ControlMaster 'yes', got %s", server.ControlMaster)
				}
				if server.ControlPath != "/tmp/ssh_control_%h_%p" {
					t.Errorf("expected ControlPath '/tmp/ssh_control_%%h_%%p', got %s", server.ControlPath)
				}
			},
		},
		{
			name: "ssh with verbose mode",
			cmd:  "ssh -v user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.LogLevel != "VERBOSE" {
					t.Errorf("expected LogLevel 'VERBOSE', got %s", server.LogLevel)
				}
			},
		},
		{
			name: "ssh with double verbose",
			cmd:  "ssh -vv user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.LogLevel != "DEBUG" {
					t.Errorf("expected LogLevel 'DEBUG', got %s", server.LogLevel)
				}
			},
		},
		{
			name: "ssh with quiet mode",
			cmd:  "ssh -q user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.LogLevel != "QUIET" {
					t.Errorf("expected LogLevel 'QUIET', got %s", server.LogLevel)
				}
			},
		},
		{
			name: "ssh with batch mode",
			cmd:  "ssh -n user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.BatchMode != sshYes {
					t.Errorf("expected BatchMode 'yes', got %s", server.BatchMode)
				}
			},
		},
		{
			name: "ssh with TTY allocation",
			cmd:  "ssh -t user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.RequestTTY != sshYes {
					t.Errorf("expected RequestTTY 'yes', got %s", server.RequestTTY)
				}
			},
		},
		{
			name: "ssh with no TTY",
			cmd:  "ssh -T user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.RequestTTY != sshNo {
					t.Errorf("expected RequestTTY 'no', got %s", server.RequestTTY)
				}
			},
		},
		{
			name: "ssh with gateway ports",
			cmd:  "ssh -g user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.GatewayPorts != sshYes {
					t.Errorf("expected GatewayPorts 'yes', got %s", server.GatewayPorts)
				}
			},
		},
		{
			name: "ssh with cipher spec",
			cmd:  "ssh -c aes256-ctr user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Ciphers != "aes256-ctr" {
					t.Errorf("expected Ciphers 'aes256-ctr', got %s", server.Ciphers)
				}
			},
		},
		{
			name: "ssh with MAC spec",
			cmd:  "ssh -m hmac-sha2-256 user@example.com",
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.MACs != "hmac-sha2-256" {
					t.Errorf("expected MACs 'hmac-sha2-256', got %s", server.MACs)
				}
			},
		},
		{
			name: "ssh without user",
			cmd:  "ssh " + testHost,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				if server.Host != testHost {
					t.Errorf("expected host '%s', got %s", testHost, server.Host)
				}
				// Smart alias generation simplifies example.com to example
				expectedAlias := "example"
				if server.Alias != expectedAlias {
					t.Errorf("expected alias '%s', got %s", expectedAlias, server.Alias)
				}
				// Default port should be 22
				if server.Port != 22 {
					t.Errorf("expected default port 22, got %d", server.Port)
				}
			},
		},
		{
			name:    "not an ssh command",
			cmd:     "ls -la",
			wantErr: true,
		},
		{
			name:    "empty command",
			cmd:     "",
			wantErr: true,
		},
		{
			name:    "ssh without host",
			cmd:     "ssh",
			wantErr: true,
		},
		{
			name:    "ssh with invalid port",
			cmd:     "ssh -p abc user@example.com",
			wantErr: true,
		},
		{
			name: "ssh with lazyssh alias comment",
			cmd: `# lazyssh-alias:myserver
ssh -p 2222 user@example.com`,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				// Should use the alias from the comment
				if server.Alias != "myserver" {
					t.Errorf("expected alias 'myserver', got %s", server.Alias)
				}
				if server.Host != testHost {
					t.Errorf("expected host '%s', got %s", testHost, server.Host)
				}
				if server.User != testUser {
					t.Errorf("expected user '%s', got %s", testUser, server.User)
				}
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
			},
		},
		{
			name: "ssh with lazyssh alias and tags comment",
			cmd: `# lazyssh-alias:prod-server tags:production,web,critical
ssh -p 443 admin@api.example.com`,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				// Should use the alias and tags from the comment
				if server.Alias != "prod-server" {
					t.Errorf("expected alias 'prod-server', got %s", server.Alias)
				}
				expectedTags := []string{"production", "web", "critical"}
				if len(server.Tags) != len(expectedTags) {
					t.Errorf("expected %d tags, got %d", len(expectedTags), len(server.Tags))
				} else {
					for i, tag := range expectedTags {
						if server.Tags[i] != tag {
							t.Errorf("expected tag[%d] = '%s', got '%s'", i, tag, server.Tags[i])
						}
					}
				}
				if server.Host != "api.example.com" {
					t.Errorf("expected host 'api.example.com', got %s", server.Host)
				}
				if server.User != "admin" {
					t.Errorf("expected user 'admin', got %s", server.User)
				}
				if server.Port != 443 {
					t.Errorf("expected port 443, got %d", server.Port)
				}
			},
		},
		{
			name: "ssh with regular comments (should be ignored)",
			cmd: `# This is a regular comment
# Another comment line
ssh -i ~/.ssh/arcsight.fia.test.pem ec2-user@arcsight.fia.test`,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				// Regular comments should be ignored
				if server.Host != "arcsight.fia.test" {
					t.Errorf("expected host 'arcsight.fia.test', got %s", server.Host)
				}
				if server.User != "ec2-user" {
					t.Errorf("expected user 'ec2-user', got %s", server.User)
				}
				if len(server.IdentityFiles) != 1 || server.IdentityFiles[0] != "~/.ssh/arcsight.fia.test.pem" {
					t.Errorf("expected identity file '~/.ssh/arcsight.fia.test.pem', got %v", server.IdentityFiles)
				}
			},
		},
		{
			name: "ssh with mixed comments and metadata",
			cmd: `# Regular comment
# lazyssh-alias:test-server tags:test
# Another regular comment
ssh user@example.com`,
			check: func(t *testing.T, s interface{}) {
				server := s.(*domain.Server)
				// Should extract metadata from lazyssh comment
				if server.Alias != "test-server" {
					t.Errorf("expected alias 'test-server', got %s", server.Alias)
				}
				if len(server.Tags) != 1 || server.Tags[0] != "test" {
					t.Errorf("expected tag 'test', got %v", server.Tags)
				}
				if server.Host != "example.com" {
					t.Errorf("expected host 'example.com', got %s", server.Host)
				}
				if server.User != "user" {
					t.Errorf("expected user 'user', got %s", server.User)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := ParseSSHCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSHCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, server)
			}
		})
	}
}

func TestPreprocessMultilineCommand(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line command",
			input: "ssh user@example.com",
			want:  "ssh user@example.com",
		},
		{
			name: "multiline with backslash",
			input: `ssh \
    -p 2222 \
    -i ~/.ssh/id_rsa \
    user@example.com`,
			want: "ssh -p 2222 -i ~/.ssh/id_rsa user@example.com",
		},
		{
			name: "multiline with options",
			input: `ssh \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -L 8080:localhost:80 \
    user@example.com`,
			want: "ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -L 8080:localhost:80 user@example.com",
		},
		{
			name: "multiline with mixed indentation",
			input: `ssh \
  -p 2222 \
        -i ~/.ssh/id_rsa \
    user@example.com`,
			want: "ssh -p 2222 -i ~/.ssh/id_rsa user@example.com",
		},
		{
			name: "multiline with empty lines",
			input: `ssh \
    -p 2222 \

    -i ~/.ssh/id_rsa \
    user@example.com`,
			want: "ssh -p 2222 -i ~/.ssh/id_rsa user@example.com",
		},
		{
			name: "complex multiline command",
			input: `ssh \
    -J bastion@jump.example.com \
    -o ProxyCommand="ssh -W %h:%p proxy" \
    -L 8080:localhost:80 \
    -L 3306:db:3306 \
    -R 9090:localhost:9090 \
    -D 1080 \
    admin@internal.example.com`,
			want: `ssh -J bastion@jump.example.com -o ProxyCommand="ssh -W %h:%p proxy" -L 8080:localhost:80 -L 3306:db:3306 -R 9090:localhost:9090 -D 1080 admin@internal.example.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessMultilineCommand(tt.input)
			if got != tt.want {
				t.Errorf("preprocessMultilineCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseSSHCommandMultiline(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		wantErr bool
		check   func(t *testing.T, server *domain.Server)
	}{
		{
			name: "multiline ssh command",
			cmd: `ssh \
    -p 2222 \
    -i ~/.ssh/id_rsa \
    user@example.com`,
			check: func(t *testing.T, server *domain.Server) {
				if server.Port != 2222 {
					t.Errorf("expected port 2222, got %d", server.Port)
				}
				if len(server.IdentityFiles) != 1 || server.IdentityFiles[0] != "~/.ssh/id_rsa" {
					t.Errorf("expected identity file '~/.ssh/id_rsa', got %v", server.IdentityFiles)
				}
				if server.User != testUser {
					t.Errorf("expected user '%s', got %s", testUser, server.User)
				}
				if server.Host != "example.com" {
					t.Errorf("expected host 'example.com', got %s", server.Host)
				}
			},
		},
		{
			name: "multiline with multiple forwards",
			cmd: `ssh \
    -L 8080:localhost:80 \
    -L 3306:db:3306 \
    -R 9090:localhost:9090 \
    -D 1080 \
    user@example.com`,
			check: func(t *testing.T, server *domain.Server) {
				if len(server.LocalForward) != 2 {
					t.Errorf("expected 2 local forwards, got %d", len(server.LocalForward))
				}
				if len(server.RemoteForward) != 1 {
					t.Errorf("expected 1 remote forward, got %d", len(server.RemoteForward))
				}
				if len(server.DynamicForward) != 1 {
					t.Errorf("expected 1 dynamic forward, got %d", len(server.DynamicForward))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := ParseSSHCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSHCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil {
				tt.check(t, server)
			}
		})
	}
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		want    []string
		wantErr bool
	}{
		{
			name: "simple command",
			cmd:  "ssh user@example.com",
			want: []string{"ssh", "user@example.com"},
		},
		{
			name: "command with quoted string",
			cmd:  `ssh user@example.com "ls -la"`,
			want: []string{"ssh", "user@example.com", "ls -la"},
		},
		{
			name: "command with single quotes",
			cmd:  `ssh user@example.com 'echo "hello world"'`,
			want: []string{"ssh", "user@example.com", `echo "hello world"`},
		},
		{
			name: "command with escaped spaces",
			cmd:  `ssh user@example.com file\ with\ spaces.txt`,
			want: []string{"ssh", "user@example.com", "file with spaces.txt"},
		},
		{
			name: "mixed quotes",
			cmd:  `ssh -o "ProxyCommand=ssh -W %h:%p bastion" user@example.com`,
			want: []string{"ssh", "-o", "ProxyCommand=ssh -W %h:%p bastion", "user@example.com"},
		},
		{
			name:    "unclosed quote",
			cmd:     `ssh user@example.com "unclosed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("splitCommand() got %d parts, want %d", len(got), len(tt.want))
					return
				}
				for i, part := range got {
					if part != tt.want[i] {
						t.Errorf("splitCommand() part[%d] = %q, want %q", i, part, tt.want[i])
					}
				}
			}
		})
	}
}
