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

package ssh_config_file

import (
	"path/filepath"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
)

func TestAddServerInsertsBeforeMatchingWildcard(t *testing.T) {
	tests := []struct {
		name   string
		config string
		alias  string
		want   string
	}{
		{
			name:   "before trailing Host *",
			config: "Include work.d/*\nUser globaluser\n\nHost a\n    HostName 10.0.0.1\n\nHost *\n    User wildcard\n",
			alias:  "b",
			want: "Include work.d/*\nUser globaluser\n\nHost a\n    HostName 10.0.0.1\n\n" +
				"Host b    #Added by lazyssh\n    HostName 10.0.0.2\n\nHost *\n    User wildcard\n",
		},
		{
			name:   "before matching pattern",
			config: "Host lab-*\n    User admin\n\nHost *\n    ForwardAgent no\n",
			alias:  "lab-3",
			want: "Host lab-3    #Added by lazyssh\n    HostName 10.0.0.2\n\n" +
				"Host lab-*\n    User admin\n\nHost *\n    ForwardAgent no\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config")
			writeFile(t, path, tt.config)

			repo := newTestRepo(t, path)
			if err := repo.AddServer(domain.Server{Alias: tt.alias, Host: "10.0.0.2"}); err != nil {
				t.Fatal(err)
			}
			if got := readFile(t, path); got != tt.want {
				t.Errorf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}
