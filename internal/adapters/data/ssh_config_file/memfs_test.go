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
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// memFS is an in-memory FileSystem implementation for tests. It implements the
// subset of the FileSystem interface our code touches; calls that need to hit
// real disk (OpenFile returning *os.File) fall through to a temp-dir backing
// so atomic-rename tests still work.
type memFS struct {
	mu       sync.Mutex
	files    map[string][]byte
	tempDir  string
	openReal map[string]string // logical path → real on-disk path for OpenFile redirects
}

func newMemFS(t interface {
	Helper()
	Fatalf(string, ...any)
},
) *memFS {
	t.Helper()
	dir, err := os.MkdirTemp("", "lazyssh-memfs-")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	return &memFS{files: map[string][]byte{}, tempDir: dir, openReal: map[string]string{}}
}

func (m *memFS) cleanup() { _ = os.RemoveAll(m.tempDir) }

func (m *memFS) write(path string, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = []byte(content)
}

func (m *memFS) read(path string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return string(m.files[path])
}

// --- FileSystem interface ---

func (m *memFS) Open(name string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.files[name]
	if !ok {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *memFS) Create(name string) (io.WriteCloser, error) {
	return &memWriter{fs: m, path: name}, nil
}

type memFileInfo struct {
	name string
	size int64
	dir  bool
	mode os.FileMode
}

func (i memFileInfo) Name() string       { return i.name }
func (i memFileInfo) Size() int64        { return i.size }
func (i memFileInfo) Mode() os.FileMode  { return i.mode }
func (i memFileInfo) ModTime() time.Time { return time.Time{} }
func (i memFileInfo) IsDir() bool        { return i.dir }
func (i memFileInfo) Sys() any           { return nil }

func (m *memFS) Stat(name string) (os.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if b, ok := m.files[name]; ok {
		return memFileInfo{name: filepath.Base(name), size: int64(len(b)), mode: 0o600}, nil
	}
	// Treat any prefix that's a parent of a known file as a directory.
	for p := range m.files {
		if strings.HasPrefix(p, name+string(os.PathSeparator)) {
			return memFileInfo{name: filepath.Base(name), dir: true, mode: 0o755 | os.ModeDir}, nil
		}
	}
	return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrNotExist}
}

func (m *memFS) IsNotExist(err error) bool { return errors.Is(err, os.ErrNotExist) }

func (m *memFS) realPathFor(logical string) string {
	if rp, ok := m.openReal[logical]; ok {
		return rp
	}
	rel := strings.ReplaceAll(filepath.Clean(logical), string(os.PathSeparator), "_")
	rp := filepath.Join(m.tempDir, rel)
	m.openReal[logical] = rp
	return rp
}

func (m *memFS) Remove(file string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[file]; ok {
		delete(m.files, file)
		return nil
	}
	if rp, ok := m.openReal[file]; ok {
		delete(m.openReal, file)
		return os.Remove(rp)
	}
	return &os.PathError{Op: "remove", Path: file, Err: os.ErrNotExist}
}

func (m *memFS) Rename(src, dst string) error {
	m.mu.Lock()
	srcReal, ok := m.openReal[src]
	m.mu.Unlock()
	if !ok {
		srcReal = src
	}
	b, err := os.ReadFile(srcReal) // #nosec G304
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.files[dst] = b
	delete(m.openReal, src)
	m.mu.Unlock()
	return os.Remove(srcReal)
}

func (m *memFS) Chmod(path string, perms os.FileMode) error { return nil }

func (m *memFS) OpenFile(path string, flag int, perms os.FileMode) (*os.File, error) {
	m.mu.Lock()
	rp := m.realPathFor(path)
	m.mu.Unlock()
	return os.OpenFile(rp, flag, perms) // #nosec G304
}

func (m *memFS) ReadDir(dir string) ([]os.DirEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var entries []os.DirEntry
	prefix := dir
	if !strings.HasSuffix(prefix, string(os.PathSeparator)) {
		prefix += string(os.PathSeparator)
	}
	for p := range m.files {
		if strings.HasPrefix(p, prefix) {
			rest := strings.TrimPrefix(p, prefix)
			if !strings.Contains(rest, string(os.PathSeparator)) {
				entries = append(entries, memDirEntry{name: rest, size: int64(len(m.files[p]))})
			}
		}
	}
	return entries, nil
}

type memDirEntry struct {
	name string
	size int64
}

func (e memDirEntry) Name() string      { return e.name }
func (e memDirEntry) IsDir() bool       { return false }
func (e memDirEntry) Type() os.FileMode { return 0 }
func (e memDirEntry) Info() (os.FileInfo, error) {
	return memFileInfo{name: e.name, size: e.size, mode: 0o600}, nil
}

type memWriter struct {
	fs   *memFS
	path string
	buf  bytes.Buffer
}

func (w *memWriter) Write(b []byte) (int, error) { return w.buf.Write(b) }
func (w *memWriter) Close() error {
	w.fs.mu.Lock()
	w.fs.files[w.path] = append([]byte(nil), w.buf.Bytes()...)
	w.fs.mu.Unlock()
	return nil
}
