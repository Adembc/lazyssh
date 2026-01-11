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

//go:build darwin

package ui

import (
	"sync"
	"time"
)

// ThemeWatcher monitors system theme changes and triggers a callback.
// On macOS, this uses polling since there's no simple CLI for theme change events.
type ThemeWatcher struct {
	stopCh    chan struct{}
	onChange  func(theme string)
	mu        sync.Mutex
	running   bool
	lastTheme string
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
	w.lastTheme = detectOSTheme()
	w.mu.Unlock()

	go w.poll()
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
}

// poll checks for theme changes every 2 seconds.
func (w *ThemeWatcher) poll() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			currentTheme := detectOSTheme()
			w.mu.Lock()
			if currentTheme != w.lastTheme {
				w.lastTheme = currentTheme
				if w.onChange != nil {
					w.mu.Unlock()
					w.onChange(currentTheme)
					continue
				}
			}
			w.mu.Unlock()
		}
	}
}
