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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLooksLikePrivateKey(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"openssh key", write("id_ed25519", "-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n"), true},
		{"pem rsa key", write("id_rsa", "-----BEGIN RSA PRIVATE KEY-----\nAAAA\n"), true},
		{"public key", write("id.pub", "ssh-ed25519 AAAA user@host\n"), false},
		{"certificate", write("cert.pem", "-----BEGIN CERTIFICATE-----\nAAAA\n"), false},
		{"known_hosts", write("known_hosts", "host ssh-ed25519 AAAA\n"), false},
		{"empty file", write("empty", ""), false},
		{"oversized file", write("big", "-----BEGIN OPENSSH PRIVATE KEY-----\n"+strings.Repeat("A", maxKeySniffSize)), false},
		{"directory", dir, false},
		{"missing", filepath.Join(dir, "nope"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikePrivateKey(tt.path); got != tt.want {
				t.Errorf("looksLikePrivateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTildePathAndExpandTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	tests := []struct {
		abs, tilde string
	}{
		{filepath.Join(home, ".ssh", "id_ed25519"), "~/.ssh/id_ed25519"},
		{home, "~"},
	}
	for _, tt := range tests {
		if got := tildePath(tt.abs); got != tt.tilde {
			t.Errorf("tildePath(%q) = %q, want %q", tt.abs, got, tt.tilde)
		}
		if got := expandTilde(tt.tilde); got != tt.abs {
			t.Errorf("expandTilde(%q) = %q, want %q", tt.tilde, got, tt.abs)
		}
	}

	outside := filepath.Join(filepath.Dir(home), "elsewhere", "key")
	if got := tildePath(outside); got != outside {
		t.Errorf("tildePath(%q) = %q, want unchanged", outside, got)
	}
	if got := expandTilde("~other/key"); got != "~other/key" {
		t.Errorf("expandTilde must not touch ~user forms, got %q", got)
	}
}

func TestMergeKeys(t *testing.T) {
	tests := []struct {
		name, current, picked string
		add                   bool
		want                  string
	}{
		{"replace", "~/.ssh/a, ~/.ssh/b", "~/.ssh/c", false, "~/.ssh/c"},
		{"add to empty", "", "~/.ssh/c", true, "~/.ssh/c"},
		{"add appends", "~/.ssh/a", "~/.ssh/c", true, "~/.ssh/a, ~/.ssh/c"},
		{"add normalizes separators", "~/.ssh/a,~/.ssh/b", "~/.ssh/c", true, "~/.ssh/a, ~/.ssh/b, ~/.ssh/c"},
		{"add skips duplicate", "~/.ssh/a, ~/.ssh/c", "~/.ssh/c", true, "~/.ssh/a, ~/.ssh/c"},
		{"add ignores empty items", "~/.ssh/a, ,", "~/.ssh/c", true, "~/.ssh/a, ~/.ssh/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeKeys(tt.current, tt.picked, tt.add); got != tt.want {
				t.Errorf("mergeKeys() = %q, want %q", got, tt.want)
			}
		})
	}
}
