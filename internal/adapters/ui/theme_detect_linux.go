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
	"os/exec"
	"strings"
)

// detectOSTheme queries the freedesktop portal for the system color scheme.
// Returns ThemeDark or ThemeLight. Defaults to ThemeDark if detection fails.
func detectOSTheme() string {
	// Query freedesktop portal: org.freedesktop.appearance color-scheme
	// Returns: 0 = no preference, 1 = prefer dark, 2 = prefer light
	cmd := exec.Command("gdbus", "call", "--session",
		"--dest", "org.freedesktop.portal.Desktop",
		"--object-path", "/org/freedesktop/portal/desktop",
		"--method", "org.freedesktop.portal.Settings.Read",
		"org.freedesktop.appearance", "color-scheme")

	output, err := cmd.Output()
	if err != nil {
		return ThemeDark // Default to dark if detection fails
	}

	// Parse output like "(<<uint32 1>>,)" or "(<<uint32 2>>,)"
	// Value 2 = prefer light, anything else = dark
	result := string(output)
	if strings.Contains(result, "uint32 2") {
		return ThemeLight
	}
	return ThemeDark
}
