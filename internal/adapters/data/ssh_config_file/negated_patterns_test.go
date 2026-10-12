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

func TestSaveKeepsNegatedPatterns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	writeFile(t, path, "Host !bastion\n    User nobody\n\nHost * !lab-? !b\n    User other\n\n")

	repo := newTestRepo(t, path)
	servers, err := repo.ListServers("")
	if err != nil {
		t.Fatal(err)
	}
	if len(servers) != 0 {
		t.Errorf("negated patterns must not be listed as servers, got %+v", servers)
	}
	if err := repo.AddServer(domain.Server{Alias: "b", Host: "10.0.0.2"}); err != nil {
		t.Fatal(err)
	}

	want := "Host !bastion\n    User nobody\n\nHost * !lab-? !b\n    User other\n\n" +
		"Host b    #Added by lazyssh\n    HostName 10.0.0.2\n"
	if got := readFile(t, path); got != want {
		t.Errorf("unexpected config:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
