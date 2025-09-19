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

func TestIsIPAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid IPv4 addresses
		{
			name:  "valid IPv4",
			input: "192.168.1.1",
			want:  true,
		},
		{
			name:  "valid IPv4 - localhost",
			input: "127.0.0.1",
			want:  true,
		},
		{
			name:  "valid IPv4 - zeros",
			input: "0.0.0.0",
			want:  true,
		},
		{
			name:  "valid IPv4 - max",
			input: "255.255.255.255",
			want:  true,
		},
		// Valid IPv6 addresses
		{
			name:  "valid IPv6 - full",
			input: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			want:  true,
		},
		{
			name:  "valid IPv6 - compressed",
			input: "2001:db8:85a3::8a2e:370:7334",
			want:  true,
		},
		{
			name:  "valid IPv6 - localhost",
			input: "::1",
			want:  true,
		},
		{
			name:  "valid IPv6 - all zeros",
			input: "::",
			want:  true,
		},
		// Invalid addresses
		{
			name:  "invalid - hostname",
			input: "example.com",
			want:  false,
		},
		{
			name:  "invalid - hostname with subdomain",
			input: "api.example.com",
			want:  false,
		},
		{
			name:  "invalid - IPv4 out of range",
			input: "256.256.256.256",
			want:  false,
		},
		{
			name:  "invalid - IPv4 with letters",
			input: "192.168.1.a",
			want:  false,
		},
		{
			name:  "invalid - too many octets",
			input: "192.168.1.1.1",
			want:  false,
		},
		{
			name:  "invalid - too few octets",
			input: "192.168.1",
			want:  false,
		},
		{
			name:  "invalid - empty string",
			input: "",
			want:  false,
		},
		{
			name:  "invalid - just dots",
			input: "...",
			want:  false,
		},
		{
			name:  "invalid - localhost name",
			input: "localhost",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsIPAddress(tt.input)
			if got != tt.want {
				t.Errorf("IsIPAddress(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
