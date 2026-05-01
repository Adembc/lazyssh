// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package ssh_config_file

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kevinburke/ssh_config"
)

const maxIncludeDepth = 16

// hasGlobMeta reports whether s contains any character special to filepath.Glob.
func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// configFile pairs a parsed ssh_config.Config with the absolute path of the
// file it came from. The path is the canonical key used everywhere CRUD
// operations need to know "which file does this host live in?".
type configFile struct {
	path string
	cfg  *ssh_config.Config
}

// loadedConfig holds the main SSH config plus every file pulled in via
// `Include` directives, in OpenSSH precedence order (main first, then includes
// depth-first in the order they appeared).
type loadedConfig struct {
	files    []configFile
	mainPath string
}

// findFile returns the configFile whose path matches absPath, or nil.
func (lc *loadedConfig) findFile(absPath string) *configFile {
	for i := range lc.files {
		if lc.files[i].path == absPath {
			return &lc.files[i]
		}
	}
	return nil
}

// paths returns the absolute paths of every loaded file in order.
func (lc *loadedConfig) paths() []string {
	out := make([]string, 0, len(lc.files))
	for _, f := range lc.files {
		out = append(out, f.path)
	}
	return out
}

// resolveIncludes parses mainPath and walks `Include` directives recursively,
// returning every file encountered. Missing globs are tolerated silently
// (matches OpenSSH). Cycles and excessive depth produce errors.
func (r *Repository) resolveIncludes(mainPath string) (*loadedConfig, error) {
	absMain, err := filepath.Abs(mainPath)
	if err != nil {
		absMain = mainPath
	}

	visited := make(map[string]bool)
	lc := &loadedConfig{mainPath: absMain}

	if err := r.loadFileAndIncludes(absMain, lc, visited, 0); err != nil {
		return nil, err
	}

	if len(lc.files) == 0 {
		// File didn't exist; preserve loadConfig's first-run behaviour.
		lc.files = append(lc.files, configFile{
			path: absMain,
			cfg:  &ssh_config.Config{Hosts: []*ssh_config.Host{}},
		})
	}
	return lc, nil
}

func (r *Repository) loadFileAndIncludes(path string, lc *loadedConfig, visited map[string]bool, depth int) error {
	if depth > maxIncludeDepth {
		return fmt.Errorf("ssh config include depth exceeded %d at %s", maxIncludeDepth, path)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	if visited[abs] {
		return fmt.Errorf("ssh config include cycle detected at %s", abs)
	}
	visited[abs] = true

	file, err := r.fileSystem.Open(abs)
	if err != nil {
		if r.fileSystem.IsNotExist(err) {
			// OpenSSH silently ignores missing includes; do the same.
			if depth == 0 {
				return nil
			}
			return nil
		}
		return fmt.Errorf("open %s: %w", abs, err)
	}

	cfg, decodeErr := ssh_config.Decode(file)
	if cerr := file.Close(); cerr != nil {
		r.logger.Warnf("failed to close %s: %v", abs, cerr)
	}
	if decodeErr != nil {
		return fmt.Errorf("decode %s: %w", abs, decodeErr)
	}

	lc.files = append(lc.files, configFile{path: abs, cfg: cfg})

	includes, err := r.parseIncludeDirectives(abs)
	if err != nil {
		return err
	}

	for _, pattern := range includes {
		expanded, err := r.expandIncludePattern(pattern)
		if err != nil {
			r.logger.Warnf("include pattern %q: %v", pattern, err)
			continue
		}
		for _, child := range expanded {
			if err := r.loadFileAndIncludes(child, lc, visited, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseIncludeDirectives scans a config file's raw text for top-level `Include`
// lines and returns their (still-unexpanded) glob patterns. We do our own
// scanning rather than relying on the parser's internal Include handling —
// kevinburke/ssh_config keeps the resolved file map unexported.
//
// Limitations (documented):
//   - Only top-level Includes are honoured. Includes inside Host/Match blocks
//     are ignored. OpenSSH allows them but they're rare; flagging as a v1 cap.
func (r *Repository) parseIncludeDirectives(absPath string) ([]string, error) {
	file, err := r.fileSystem.Open(absPath)
	if err != nil {
		if r.fileSystem.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("re-open %s for include scan: %w", absPath, err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			r.logger.Warnf("failed to close %s during include scan: %v", absPath, cerr)
		}
	}()

	var patterns []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	inHostOrMatch := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, rest := splitDirective(trimmed)
		if key == "" {
			continue
		}
		lower := strings.ToLower(key)

		switch lower {
		case "host", "match":
			inHostOrMatch = true
			continue
		case "include":
			if inHostOrMatch {
				continue
			}
			patterns = append(patterns, splitIncludeArgs(rest)...)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", absPath, err)
	}
	return patterns, nil
}

// splitDirective splits "Key value..." (or "Key=value...") into (key, rest).
func splitDirective(line string) (string, string) {
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == ' ' || c == '\t' || c == '=' {
			key := line[:i]
			rest := strings.TrimLeft(line[i:], " \t=")
			return key, rest
		}
	}
	return line, ""
}

// splitIncludeArgs splits the argument list of an `Include` directive,
// respecting double-quoted patterns (which may contain spaces).
func splitIncludeArgs(rest string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch {
		case c == '"':
			inQuote = !inQuote
		case (c == ' ' || c == '\t') && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// expandIncludePattern resolves a single `Include` pattern: handles `~/`
// expansion, glob-expands, and resolves relative paths against `~/.ssh/`
// (matching OpenSSH semantics).
func (r *Repository) expandIncludePattern(pattern string) ([]string, error) {
	if pattern == "" {
		return nil, nil
	}

	expanded := pattern
	if strings.HasPrefix(expanded, "~/") || expanded == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve ~ for include %q: %w", pattern, err)
		}
		if expanded == "~" {
			expanded = home
		} else {
			expanded = filepath.Join(home, expanded[2:])
		}
	}

	if !filepath.IsAbs(expanded) {
		// Relative paths in user config resolve against ~/.ssh.
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve relative include %q: %w", pattern, err)
		}
		expanded = filepath.Join(home, ".ssh", expanded)
	}

	var matches []string
	if hasGlobMeta(expanded) {
		m, err := filepath.Glob(expanded)
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", expanded, err)
		}
		matches = m
	} else {
		// Literal path — go through the FileSystem abstraction so tests using
		// an in-memory FS still resolve.
		if _, err := r.fileSystem.Stat(expanded); err == nil {
			matches = []string{expanded}
		} else if !r.fileSystem.IsNotExist(err) {
			return nil, fmt.Errorf("stat %q: %w", expanded, err)
		}
	}
	sort.Strings(matches)

	resolved := make([]string, 0, len(matches))
	for _, m := range matches {
		// Skip directories — OpenSSH only Includes regular files.
		info, statErr := r.fileSystem.Stat(m)
		if statErr != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		abs, err := filepath.Abs(m)
		if err != nil {
			abs = m
		}
		resolved = append(resolved, abs)
	}
	return resolved, nil
}
