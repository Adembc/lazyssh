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
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func newRepoForFS(t *testing.T, fs *memFS, metaPath string) *Repository {
	t.Helper()
	logger := zap.NewNop().Sugar()
	return &Repository{
		logger:          logger,
		configPath:      "/home/u/.ssh/config",
		fileSystem:      fs,
		metadataManager: newMetadataManager(metaPath, logger),
	}
}

func TestUpdateServer_AmbiguousAcrossFiles(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	work := "/home/u/.ssh/work"
	personal := "/home/u/.ssh/personal"
	fs.write(main, "Include "+work+"\nInclude "+personal+"\n")
	fs.write(work, "Host shared\n  HostName work.example.com\n")
	fs.write(personal, "Host shared\n  HostName home.example.com\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	srv := domain.Server{Alias: "shared", Host: "work.example.com", User: "u"}
	newSrv := srv
	newSrv.User = "ubuntu"

	err := r.UpdateServer(srv, newSrv)
	var ambig *domain.ErrAmbiguousHost
	if !errors.As(err, &ambig) {
		t.Fatalf("want ErrAmbiguousHost, got %v", err)
	}
	if ambig.Alias != "shared" {
		t.Errorf("alias = %q", ambig.Alias)
	}
	if len(ambig.Candidates) != 2 {
		t.Fatalf("want 2 candidates, got %v", ambig.Candidates)
	}

	// Re-invoke with explicit SourceFile picks the right file.
	srv2 := srv
	srv2.SourceFile = personal
	newSrv2 := newSrv
	newSrv2.SourceFile = personal
	if err := r.UpdateServer(srv2, newSrv2); err != nil {
		t.Fatalf("update with SourceFile: %v", err)
	}

	personalContent := fs.read(personal)
	if !strings.Contains(personalContent, "User ubuntu") {
		t.Errorf("personal file missing update: %s", personalContent)
	}
	workContent := fs.read(work)
	if strings.Contains(workContent, "User ubuntu") {
		t.Errorf("work file should be untouched: %s", workContent)
	}
}

func TestUpdateServer_RoutesToOwningFile(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	work := "/home/u/.ssh/work"
	fs.write(main, "Include "+work+"\nHost local\n  HostName 127.0.0.1\n")
	fs.write(work, "Host prod\n  HostName prod.example.com\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	srv := domain.Server{Alias: "prod", Host: "prod.example.com", User: ""}
	newSrv := srv
	newSrv.User = "deploy"

	if err := r.UpdateServer(srv, newSrv); err != nil {
		t.Fatalf("update: %v", err)
	}

	workContent := fs.read(work)
	if !strings.Contains(workContent, "User deploy") {
		t.Errorf("work file missing update: %s", workContent)
	}
	mainContent := fs.read(main)
	if strings.Contains(mainContent, "User deploy") {
		t.Errorf("main file should be untouched")
	}
}

func TestAddServer_DefaultsToMainFile(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	work := "/home/u/.ssh/work"
	fs.write(main, "Include "+work+"\n")
	fs.write(work, "Host existing\n  HostName 1.1.1.1\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	srv := domain.Server{Alias: "fresh", Host: "fresh.example.com", User: "u"}
	if err := r.AddServer(srv); err != nil {
		t.Fatalf("add: %v", err)
	}

	mainContent := fs.read(main)
	if !strings.Contains(mainContent, "Host fresh") {
		t.Errorf("new host should be in main file: %s", mainContent)
	}
	workContent := fs.read(work)
	if strings.Contains(workContent, "Host fresh") {
		t.Errorf("new host should not be in include file")
	}
}

func TestDeleteServer_AmbiguousReturnsErr(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	work := "/home/u/.ssh/work"
	fs.write(main, "Include "+work+"\nHost shared\n  HostName a\n")
	fs.write(work, "Host shared\n  HostName b\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	err := r.DeleteServer(domain.Server{Alias: "shared"})
	var ambig *domain.ErrAmbiguousHost
	if !errors.As(err, &ambig) {
		t.Fatalf("want ErrAmbiguousHost, got %v", err)
	}
}

func TestUpdateServer_PersistsFileChoiceToMetadata(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	work := "/home/u/.ssh/work"
	fs.write(main, "Include "+work+"\n")
	fs.write(work, "Host pinned\n  HostName 1.1.1.1\n")

	tmpMeta := filepath.Join(t.TempDir(), "metadata.json")
	r := newRepoForFS(t, fs, tmpMeta)

	srv := domain.Server{Alias: "pinned", Host: "1.1.1.1"}
	newSrv := srv
	newSrv.User = "deploy"
	if err := r.UpdateServer(srv, newSrv); err != nil {
		t.Fatalf("update: %v", err)
	}

	meta, err := r.metadataManager.loadAll()
	if err != nil {
		t.Fatalf("loadAll: %v", err)
	}
	if got := meta["pinned"].File; got != work {
		t.Errorf("metadata.File = %q, want %q", got, work)
	}
}
