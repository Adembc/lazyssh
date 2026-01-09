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
	"os/exec"
	"strings"
)

// detectOSTheme queries macOS for the system appearance.
// Returns ThemeDark or ThemeLight. Defaults to ThemeLight if detection fails
// (macOS defaults to light mode when AppleInterfaceStyle is not set).
func detectOSTheme() string {
	// defaults read -g AppleInterfaceStyle returns "Dark" if dark mode
	// If light mode, the key doesn't exist and command returns error
	cmd := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle")

	output, err := cmd.Output()
	if err != nil {
		// Key not found = light mode (macOS default)
		return ThemeLight
	}

	result := strings.TrimSpace(string(output))
	if strings.EqualFold(result, "Dark") {
		return ThemeDark
	}
	return ThemeLight
}
