// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package ssh_config_file

import (
	"strings"
	"testing"

	"go.uber.org/zap"
)

func newTestRepo(t *testing.T, fs FileSystem, mainPath string) *Repository {
	t.Helper()
	logger := zap.NewNop().Sugar()
	return &Repository{
		logger:          logger,
		configPath:      mainPath,
		fileSystem:      fs,
		metadataManager: newMetadataManager("/dev/null", logger),
	}
}

func TestResolveIncludes_NoIncludes(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "Host alpha\n  HostName 1.1.1.1\n")

	r := newTestRepo(t, fs, main)
	lc, err := r.resolveIncludes(main)
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}
	if len(lc.files) != 1 {
		t.Fatalf("want 1 file, got %d", len(lc.files))
	}
	if lc.files[0].path != main {
		t.Errorf("want path %s, got %s", main, lc.files[0].path)
	}
}

func TestResolveIncludes_AbsoluteInclude(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	inc := "/home/u/.ssh/work"
	fs.write(main, "Include "+inc+"\nHost alpha\n  HostName 1.1.1.1\n")
	fs.write(inc, "Host beta\n  HostName 2.2.2.2\n")

	r := newTestRepo(t, fs, main)
	lc, err := r.resolveIncludes(main)
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}
	if len(lc.files) != 2 {
		t.Fatalf("want 2 files, got %d (%v)", len(lc.files), lc.paths())
	}
	if lc.files[0].path != main || lc.files[1].path != inc {
		t.Errorf("ordering wrong: %v", lc.paths())
	}
}

func TestResolveIncludes_MissingIncludeIsSilent(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	fs.write(main, "Include /nonexistent/file\nHost a\n  HostName x\n")

	r := newTestRepo(t, fs, main)
	lc, err := r.resolveIncludes(main)
	if err != nil {
		t.Fatalf("missing include should not error: %v", err)
	}
	if len(lc.files) != 1 {
		t.Errorf("want 1 file, got %d", len(lc.files))
	}
}

func TestResolveIncludes_CycleDetected(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	a := "/home/u/.ssh/config"
	b := "/home/u/.ssh/loop"
	fs.write(a, "Include "+b+"\n")
	fs.write(b, "Include "+a+"\n")

	r := newTestRepo(t, fs, a)
	_, err := r.resolveIncludes(a)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("want cycle error, got %v", err)
	}
}

func TestResolveIncludes_IgnoresIncludeInsideHostBlock(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	inc := "/home/u/.ssh/should-not-load"
	fs.write(main, "Host alpha\n  HostName 1.1.1.1\n  Include "+inc+"\n")
	fs.write(inc, "Host beta\n  HostName 2.2.2.2\n")

	r := newTestRepo(t, fs, main)
	lc, err := r.resolveIncludes(main)
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}
	if len(lc.files) != 1 {
		t.Errorf("Include inside Host block should be ignored, got files=%v", lc.paths())
	}
}

func TestResolveIncludes_FirstFileMissing(t *testing.T) {
	fs := newMemFS(t)
	defer fs.cleanup()

	main := "/home/u/.ssh/config"
	r := newTestRepo(t, fs, main)
	lc, err := r.resolveIncludes(main)
	if err != nil {
		t.Fatalf("resolveIncludes: %v", err)
	}
	if len(lc.files) != 1 {
		t.Fatalf("want 1 placeholder file, got %d", len(lc.files))
	}
	if got := len(lc.files[0].cfg.Hosts); got != 0 {
		t.Errorf("placeholder cfg should be empty, got %d hosts", got)
	}
}

func TestSplitIncludeArgs(t *testing.T) {
	got := splitIncludeArgs(`config.d/* "with spaces.conf" plain`)
	want := []string{"config.d/*", "with spaces.conf", "plain"}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d: got %q want %q", i, got[i], want[i])
		}
	}
}
