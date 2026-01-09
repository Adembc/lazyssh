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

package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestSetTheme(t *testing.T) {
	tests := []struct {
		name            string
		themeName       string
		expectedMode    string
		expectedThemeIn []string // CurrentTheme.Name should be one of these
	}{
		{
			name:            "set dark theme",
			themeName:       ThemeDark,
			expectedMode:    ThemeDark,
			expectedThemeIn: []string{ThemeDark},
		},
		{
			name:            "set light theme",
			themeName:       ThemeLight,
			expectedMode:    ThemeLight,
			expectedThemeIn: []string{ThemeLight},
		},
		{
			name:            "set system theme",
			themeName:       ThemeSystem,
			expectedMode:    ThemeSystem,
			expectedThemeIn: []string{ThemeDark, ThemeLight}, // depends on OS
		},
		{
			name:            "unknown theme defaults to dark",
			themeName:       "unknown",
			expectedMode:    ThemeDark,
			expectedThemeIn: []string{ThemeDark},
		},
		{
			name:            "empty string defaults to dark",
			themeName:       "",
			expectedMode:    ThemeDark,
			expectedThemeIn: []string{ThemeDark},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTheme(tt.themeName)
			if CurrentThemeMode != tt.expectedMode {
				t.Errorf("SetTheme(%q) mode = %q, want %q", tt.themeName, CurrentThemeMode, tt.expectedMode)
			}
			found := false
			for _, expected := range tt.expectedThemeIn {
				if CurrentTheme.Name == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("SetTheme(%q) theme = %q, want one of %v", tt.themeName, CurrentTheme.Name, tt.expectedThemeIn)
			}
		})
	}

	// Reset to dark theme after tests
	SetTheme(ThemeDark)
}

func TestGetThemeNames(t *testing.T) {
	names := GetThemeNames()

	if len(names) != 3 {
		t.Errorf("GetThemeNames() returned %d themes, want 3", len(names))
	}

	expectedNames := map[string]bool{ThemeDark: true, ThemeLight: true, ThemeSystem: true}
	for _, name := range names {
		if !expectedNames[name] {
			t.Errorf("GetThemeNames() contains unexpected theme %q", name)
		}
	}
}

func TestDarkThemeColors(t *testing.T) {
	// Verify dark theme has reasonable color values
	theme := DarkTheme

	if theme.Name != ThemeDark {
		t.Errorf("DarkTheme.Name = %q, want %q", theme.Name, ThemeDark)
	}

	// Check that background is dark (low color number = dark)
	if theme.PrimitiveBackground > tcell.Color240 {
		t.Errorf("DarkTheme.PrimitiveBackground should be a dark color, got %v", theme.PrimitiveBackground)
	}

	// Check hex colors are not empty
	hexColors := []struct {
		name  string
		value string
	}{
		{"BrandPrimary", theme.BrandPrimary},
		{"BrandSecondary", theme.BrandSecondary},
		{"StatusSuccess", theme.StatusSuccess},
		{"StatusError", theme.StatusError},
	}

	for _, hc := range hexColors {
		if hc.value == "" {
			t.Errorf("DarkTheme.%s should not be empty", hc.name)
		}
	}
}

func TestLightThemeColors(t *testing.T) {
	// Verify light theme has reasonable color values
	theme := LightTheme

	if theme.Name != ThemeLight {
		t.Errorf("LightTheme.Name = %q, want %q", theme.Name, ThemeLight)
	}

	// Check that background is light (high color number = light)
	if theme.PrimitiveBackground < tcell.Color240 {
		t.Errorf("LightTheme.PrimitiveBackground should be a light color, got %v", theme.PrimitiveBackground)
	}

	// Check hex colors are not empty
	hexColors := []struct {
		name  string
		value string
	}{
		{"BrandPrimary", theme.BrandPrimary},
		{"BrandSecondary", theme.BrandSecondary},
		{"StatusSuccess", theme.StatusSuccess},
		{"StatusError", theme.StatusError},
	}

	for _, hc := range hexColors {
		if hc.value == "" {
			t.Errorf("LightTheme.%s should not be empty", hc.name)
		}
	}
}

func TestThemesHaveDifferentColors(t *testing.T) {
	// Ensure dark and light themes are actually different
	if DarkTheme.PrimitiveBackground == LightTheme.PrimitiveBackground {
		t.Error("DarkTheme and LightTheme should have different PrimitiveBackground colors")
	}

	if DarkTheme.BrandPrimary == LightTheme.BrandPrimary {
		t.Error("DarkTheme and LightTheme should have different BrandPrimary colors")
	}

	if DarkTheme.MutedText == LightTheme.MutedText {
		t.Error("DarkTheme and LightTheme should have different MutedText colors")
	}
}
