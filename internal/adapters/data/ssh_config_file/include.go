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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/kevinburke/ssh_config"
)

// loadAllServers loads servers from the main ssh config file and recursively
// from non-commented Include directives, returning a flat, de-duplicated list.
// Main config takes precedence on alias conflicts.
func (r *Repository) loadAllServers() ([]domain.Server, error) { //nolint:unparam // kept for symmetry and future enhancements
	mainPath := expandTilde(r.configPath)
	absMain, err := filepath.Abs(mainPath)
	if err != nil {
		absMain = mainPath
	}

	files := []string{absMain}
	visited := map[string]struct{}{absMain: {}}

	included, err := r.resolveIncludes(absMain, visited)
	if err != nil {
		r.logger.Warnf("failed to resolve includes: %v", err)
	}
	files = append(files, included...)

	seen := make(map[string]struct{}, 64)
	all := make([]domain.Server, 0, 64)

	for i, f := range files {
		cfg, err := r.decodeConfigAt(f)
		if err != nil {
			r.logger.Warnf("failed to decode %s: %v", f, err)
			continue
		}
		isMain := i == 0
		servers := r.toDomainServersFromConfig(cfg, f, isMain)
		for _, s := range servers {
			if _, ok := seen[s.Alias]; ok {
				continue
			}
			seen[s.Alias] = struct{}{}
			all = append(all, s)
		}
	}
	return all, nil
}

// resolveIncludes parses a config file for non-commented Include directives,
// supports multiple patterns per line, globs, tilde-expansion and relative paths.
// It returns a depth-first ordered list of unique absolute file paths.
func (r *Repository) resolveIncludes(filePath string, visited map[string]struct{}) ([]string, error) {
	fp := expandTilde(filePath)
	if !filepath.IsAbs(fp) {
		if ap, err := filepath.Abs(fp); err == nil {
			fp = ap
		}
	}

	f, err := r.fileSystem.Open(fp)
	if err != nil {
		// If the including file can't be read, treat as no includes
		return []string{}, nil
	}
	defer func() {
		_ = f.Close()
	}()

	baseDir := filepath.Dir(fp)
	results := make([]string, 0)
	added := make(map[string]struct{})

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip inline comments (only when '#' is not in quotes)
		line = stripInlineComment(line)
		if line == "" {
			continue
		}
		// skip commented lines
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Tokenize respecting quotes
		fields := splitFieldsRespectQuotes(line)
		if len(fields) == 0 {
			continue
		}

		// Look for "Include"
		if !strings.EqualFold(fields[0], "Include") {
			continue
		}
		patterns := fields[1:]
		for _, pat := range patterns {
			p := unquote(strings.TrimSpace(pat))
			if p == "" {
				continue
			}
			p = expandTilde(p)
			if !filepath.IsAbs(p) {
				p = filepath.Join(baseDir, p)
			}
			globbed, gerr := filepath.Glob(p)
			if gerr != nil || len(globbed) == 0 {
				// OpenSSH ignores unmatched includes
				continue
			}
			for _, m := range globbed {
				child := m
				if !filepath.IsAbs(child) {
					if ap, err := filepath.Abs(child); err == nil {
						child = ap
					}
				}
				if _, ok := visited[child]; ok {
					continue
				}
				visited[child] = struct{}{}
				if _, ok := added[child]; !ok {
					results = append(results, child)
					added[child] = struct{}{}
				}
				// Recurse
				sub, _ := r.resolveIncludes(child, visited)
				for _, s := range sub {
					if _, ok := added[s]; !ok {
						results = append(results, s)
						added[s] = struct{}{}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return results, fmt.Errorf("scanner error: %w", err)
	}
	return results, nil
}

// decodeConfigAt decodes a single ssh config file at the given absolute path.
func (r *Repository) decodeConfigAt(path string) (*ssh_config.Config, error) {
	rc, err := r.fileSystem.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	return ssh_config.Decode(rc)
}

func expandTilde(p string) string {
	if p == "" {
		return p
	}
	if p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	// We don't support ~user syntax; return as-is
	return p
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// stripInlineComment removes an unquoted '#' and everything after it.
func stripInlineComment(s string) string {
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return strings.TrimSpace(s[:i])
			}
		}
	}
	return strings.TrimSpace(s)
}

// splitFieldsRespectQuotes splits a line into fields while preserving quoted segments.
func splitFieldsRespectQuotes(s string) []string {
	var fields []string
	var b strings.Builder
	inSingle := false
	inDouble := false

	flush := func() {
		if b.Len() > 0 {
			fields = append(fields, b.String())
			b.Reset()
		}
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch ch {
		case ' ', '\t':
			if inSingle || inDouble {
				b.WriteByte(ch)
			} else {
				flush()
			}
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			b.WriteByte(ch)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	flush()
	return fields
}
