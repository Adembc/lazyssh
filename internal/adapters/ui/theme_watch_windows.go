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

//go:build windows

package ui

// ThemeWatcher monitors system theme changes and triggers a callback.
// On Windows, live theme watching is not yet implemented.
type ThemeWatcher struct {
	onChange func(theme string)
}

// NewThemeWatcher creates a new theme watcher with the given callback.
func NewThemeWatcher(onChange func(theme string)) *ThemeWatcher {
	return &ThemeWatcher{onChange: onChange}
}

// Start begins watching for system theme changes.
// On Windows, this is a no-op (live watching not yet implemented).
func (w *ThemeWatcher) Start() {}

// Stop stops watching for theme changes.
func (w *ThemeWatcher) Stop() {}
