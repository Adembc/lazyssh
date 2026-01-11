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

//go:build linux

package ui

import (
	"bufio"
	"os/exec"
	"strings"
	"sync"
)

// ThemeWatcher monitors system theme changes and triggers a callback.
type ThemeWatcher struct {
	cmd      *exec.Cmd
	stopCh   chan struct{}
	onChange func(theme string)
	mu       sync.Mutex
	running  bool
}

// NewThemeWatcher creates a new theme watcher with the given callback.
func NewThemeWatcher(onChange func(theme string)) *ThemeWatcher {
	return &ThemeWatcher{
		onChange: onChange,
		stopCh:   make(chan struct{}),
	}
}

// Start begins watching for system theme changes.
func (w *ThemeWatcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	go w.watch()
}

// Stop stops watching for theme changes.
func (w *ThemeWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}
	w.running = false
	close(w.stopCh)

	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		w.cmd = nil
	}
}

// watch monitors the freedesktop portal for color-scheme changes.
func (w *ThemeWatcher) watch() {
	// Monitor the settings changed signal from the freedesktop portal
	w.cmd = exec.Command("gdbus", "monitor", "--session",
		"--dest", "org.freedesktop.portal.Desktop",
		"--object-path", "/org/freedesktop/portal/desktop")

	stdout, err := w.cmd.StdoutPipe()
	if err != nil {
		return
	}

	if err := w.cmd.Start(); err != nil {
		return
	}

	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			select {
			case <-w.stopCh:
				return
			default:
				line := scanner.Text()
				// Look for SettingChanged signal with color-scheme
				if strings.Contains(line, "SettingChanged") &&
					strings.Contains(line, "org.freedesktop.appearance") &&
					strings.Contains(line, "color-scheme") {
					// Detect the new theme
					newTheme := detectOSTheme()
					if w.onChange != nil {
						w.onChange(newTheme)
					}
				}
			}
		}
	}()

	// Wait for stop signal
	<-w.stopCh
	if w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
	}
	_ = w.cmd.Wait()
}
