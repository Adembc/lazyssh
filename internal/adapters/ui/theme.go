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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Theme name constants.
const (
	ThemeDark   = "dark"
	ThemeLight  = "light"
	ThemeSystem = "system"
)

// Theme defines all colors used throughout the application.
// It contains both tcell.Color values for component styling and
// hex color strings for tview markup-based dynamic coloring.
type Theme struct {
	Name string

	// tview.Styles colors (applied globally via ApplyTheme)
	PrimitiveBackground tcell.Color
	ContrastBackground  tcell.Color
	BorderColor         tcell.Color
	TitleColor          tcell.Color
	PrimaryText         tcell.Color
	SecondaryText       tcell.Color
	GraphicsColor       tcell.Color

	// Component-specific tcell colors
	HeaderBackground    tcell.Color
	SearchFieldBg       tcell.Color
	SearchFieldText     tcell.Color
	SelectedBackground  tcell.Color
	SelectedText        tcell.Color
	StatusBarBackground tcell.Color

	// Hex colors for tview markup (used in SetText with dynamic colors)
	BrandPrimary     string // "lazy" text in header
	BrandSecondary   string // "ssh" text in header
	VersionTag       string // version chip background
	CommitTag        string // commit chip background
	LinkColor        string // clickable links
	MutedText        string // secondary/muted text
	DimText          string // tertiary/dim text
	Separator        string // separator lines
	TagChipBg        string // tag chip background
	TagChipText      string // tag chip text
	TagExtra         string // "+N" extra tags indicator
	ForwardingActive string // active forwarding indicator
	StatusSuccess    string // success status messages
	StatusError      string // error status messages
	HintKey          string // keyboard hint keys
}

// DarkTheme is the default dark color scheme.
var DarkTheme = Theme{
	Name: ThemeDark,

	// tview.Styles colors
	PrimitiveBackground: tcell.Color232,
	ContrastBackground:  tcell.Color235,
	BorderColor:         tcell.Color238,
	TitleColor:          tcell.Color250,
	PrimaryText:         tcell.Color252,
	SecondaryText:       tcell.Color245,
	GraphicsColor:       tcell.Color238,

	// Component-specific colors
	HeaderBackground:    tcell.Color234,
	SearchFieldBg:       tcell.Color233,
	SearchFieldText:     tcell.Color252,
	SelectedBackground:  tcell.Color24,
	SelectedText:        tcell.Color255,
	StatusBarBackground: tcell.Color235,

	// Hex colors for markup
	BrandPrimary:     "#FFFFFF",
	BrandSecondary:   "#55D7FF",
	VersionTag:       "#22C55E",
	CommitTag:        "#A78BFA",
	LinkColor:        "#55AAFF",
	MutedText:        "#AAAAAA",
	DimText:          "#888888",
	Separator:        "#444444",
	TagChipBg:        "#5FAFFF",
	TagChipText:      "black",
	TagExtra:         "#8A8A8A",
	ForwardingActive: "#A0FFA0",
	StatusSuccess:    "#A0FFA0",
	StatusError:      "#FF6B6B",
	HintKey:          "white",
}

// LightTheme is the light color scheme for better visibility in bright environments.
var LightTheme = Theme{
	Name: ThemeLight,

	// tview.Styles colors
	PrimitiveBackground: tcell.Color255,
	ContrastBackground:  tcell.Color254,
	BorderColor:         tcell.Color245,
	TitleColor:          tcell.Color236,
	PrimaryText:         tcell.Color232,
	SecondaryText:       tcell.Color240,
	GraphicsColor:       tcell.Color245,

	// Component-specific colors
	HeaderBackground:    tcell.Color253,
	SearchFieldBg:       tcell.Color254,
	SearchFieldText:     tcell.Color232,
	SelectedBackground:  tcell.Color39,
	SelectedText:        tcell.Color232,
	StatusBarBackground: tcell.Color254,

	// Hex colors for markup
	BrandPrimary:     "#1A1A1A",
	BrandSecondary:   "#0088CC",
	VersionTag:       "#16A34A",
	CommitTag:        "#7C3AED",
	LinkColor:        "#0066CC",
	MutedText:        "#666666",
	DimText:          "#888888",
	Separator:        "#CCCCCC",
	TagChipBg:        "#0088CC",
	TagChipText:      "white",
	TagExtra:         "#666666",
	ForwardingActive: "#16A34A",
	StatusSuccess:    "#16A34A",
	StatusError:      "#DC2626",
	HintKey:          "#1A1A1A",
}

// CurrentTheme is the active theme used throughout the application.
// It defaults to DarkTheme and can be changed via SetTheme.
var CurrentTheme = &DarkTheme

// CurrentThemeMode tracks the user's theme selection (dark, light, or system).
// This may differ from CurrentTheme.Name when system theme is selected.
var CurrentThemeMode = ThemeDark

// SetTheme sets the current theme by name.
// Valid names are ThemeDark, ThemeLight, and ThemeSystem.
// ThemeSystem detects the OS preference. Unknown names default to dark.
func SetTheme(name string) {
	CurrentThemeMode = name
	switch name {
	case ThemeLight:
		CurrentTheme = &LightTheme
	case ThemeSystem:
		// Detect OS preference and apply appropriate theme
		detected := detectOSTheme()
		if detected == ThemeLight {
			CurrentTheme = &LightTheme
		} else {
			CurrentTheme = &DarkTheme
		}
	default:
		CurrentThemeMode = ThemeDark
		CurrentTheme = &DarkTheme
	}
}

// ApplyTheme applies the current theme to tview's global Styles.
// This should be called once during initialization, before any UI components are created.
func ApplyTheme() {
	tview.Styles.PrimitiveBackgroundColor = CurrentTheme.PrimitiveBackground
	tview.Styles.ContrastBackgroundColor = CurrentTheme.ContrastBackground
	tview.Styles.BorderColor = CurrentTheme.BorderColor
	tview.Styles.TitleColor = CurrentTheme.TitleColor
	tview.Styles.PrimaryTextColor = CurrentTheme.PrimaryText
	tview.Styles.SecondaryTextColor = CurrentTheme.SecondaryText
	tview.Styles.TertiaryTextColor = CurrentTheme.SecondaryText
	tview.Styles.GraphicsColor = CurrentTheme.GraphicsColor
}

// GetThemeNames returns the list of available theme names.
func GetThemeNames() []string {
	return []string{ThemeDark, ThemeLight, ThemeSystem}
}
