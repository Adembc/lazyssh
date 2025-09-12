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
)

func TestParseSSHCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		want    ServerFormData
		wantErr bool
	}{
		{
			name: "basic ssh user@host",
			cmd:  "ssh user@example.com",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "22",
				Key:   "~/.ssh/id_ed25519",
			},
		},
		{
			name: "ssh with port",
			cmd:  "ssh user@example.com -p 2222",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "2222",
				Key:   "~/.ssh/id_ed25519",
			},
		},
		{
			name: "ssh with key",
			cmd:  "ssh user@example.com -i ~/.ssh/custom_key",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "22",
				Key:   "~/.ssh/custom_key",
			},
		},
		{
			name: "ssh with port and key",
			cmd:  "ssh user@example.com -p 2222 -i ~/.ssh/custom_key",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "2222",
				Key:   "~/.ssh/custom_key",
			},
		},
		{
			name: "ssh with options before host",
			cmd:  "ssh -p 2222 -i ~/.ssh/custom_key user@example.com",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "2222",
				Key:   "~/.ssh/custom_key",
			},
		},
		{
			name: "ssh host only",
			cmd:  "ssh example.com",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "root",
				Port:  "22",
				Key:   "~/.ssh/id_ed25519",
			},
		},
		{
			name: "without ssh prefix",
			cmd:  "user@example.com",
			want: ServerFormData{
				Alias: "example",
				Host:  "example.com",
				User:  "user",
				Port:  "22",
				Key:   "~/.ssh/id_ed25519",
			},
		},
		{
			name: "IP address",
			cmd:  "ssh root@192.168.1.100",
			want: ServerFormData{
				Alias: "192.168.1.100",
				Host:  "192.168.1.100",
				User:  "root",
				Port:  "22",
				Key:   "~/.ssh/id_ed25519",
			},
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
			name:    "invalid port",
			cmd:     "ssh user@example.com -p abc",
			wantErr: true,
		},
		{
			name:    "port out of range",
			cmd:     "ssh user@example.com -p 70000",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSSHCommand(tt.cmd)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSSHCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Alias != tt.want.Alias {
					t.Errorf("ParseSSHCommand() Alias = %v, want %v", got.Alias, tt.want.Alias)
				}
				if got.Host != tt.want.Host {
					t.Errorf("ParseSSHCommand() Host = %v, want %v", got.Host, tt.want.Host)
				}
				if got.User != tt.want.User {
					t.Errorf("ParseSSHCommand() User = %v, want %v", got.User, tt.want.User)
				}
				if got.Port != tt.want.Port {
					t.Errorf("ParseSSHCommand() Port = %v, want %v", got.Port, tt.want.Port)
				}
				if got.Key != tt.want.Key {
					t.Errorf("ParseSSHCommand() Key = %v, want %v", got.Key, tt.want.Key)
				}
			}
		})
	}
}
