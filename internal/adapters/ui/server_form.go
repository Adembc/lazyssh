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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// sshDefaults is now replaced by SSHFieldDefaults in defaults.go
// This variable references the centralized defaults for consistency
var sshDefaults = SSHFieldDefaults

type ServerFormMode int

const (
	ServerFormAdd ServerFormMode = iota
	ServerFormEdit
)

const (
	tabSeparator = "[gray]|[-] " // Tab separator with gray color

	// SessionType values (sessionTypeNone and sessionTypeSubsystem are in utils.go)
	sessionTypeDefault = "default"
)

type ServerForm struct {
	*tview.Flex                 // The root container (includes header, form panel and hint bar)
	header          *AppHeader  // The app header
	formPanel       *tview.Flex // The actual form panel
	pages           *tview.Pages
	tabBar          *tview.TextView
	forms           map[string]*tview.Form
	currentTab      string
	tabs            []string
	tabAbbrev       map[string]string // Abbreviated tab names for narrow views
	mode            ServerFormMode
	original        *domain.Server
	initialData     *domain.Server // Initial data for pre-filling form (used in Add mode)
	existingAliases []string       // List of existing aliases for validation
	onSave          func(domain.Server, *domain.Server)
	onCancel        func()
	app             *tview.Application // Reference to app for showing modals
	version         string             // Version for header
	commit          string             // Commit for header
	validation      *ValidationState   // Validation state for all fields
	helpPanel       *tview.TextView    // Help panel for field descriptions
	helpMode        HelpDisplayMode    // Current help display mode
	currentField    string             // Currently focused field
	mainContainer   *tview.Flex        // Container for form and help panel
}

func NewServerForm(mode ServerFormMode, original *domain.Server) *ServerForm {
	// Create help panel
	helpPanel := tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetScrollable(true)
	helpPanel.SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle(" Help ").
		SetTitleAlign(tview.AlignCenter)

	// Create main container for form and help
	mainContainer := tview.NewFlex().SetDirection(tview.FlexColumn)

	form := &ServerForm{
		Flex:          tview.NewFlex().SetDirection(tview.FlexRow),
		formPanel:     tview.NewFlex().SetDirection(tview.FlexRow),
		pages:         tview.NewPages(),
		tabBar:        tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).SetRegions(true),
		forms:         make(map[string]*tview.Form),
		mode:          mode,
		original:      original,
		validation:    NewValidationState(),
		helpPanel:     helpPanel,
		helpMode:      HelpModeNormal, // Show help panel by default
		mainContainer: mainContainer,
		tabs: []string{
			"Basic",
			"Connection",
			"Forwarding",
			"Authentication",
			"Advanced",
		},
		tabAbbrev: map[string]string{
			"Basic":          "Basic",
			"Connection":     "Conn",
			"Forwarding":     "Fwd",
			"Authentication": "Auth",
			"Advanced":       "Adv",
		},
	}
	form.currentTab = "Basic"
	// Don't build here, wait for version info to be set
	return form
}

func (sf *ServerForm) build() {
	// Create header
	sf.header = NewAppHeader(sf.version, sf.commit, RepoURL)

	// Create forms for each tab
	sf.createBasicForm()
	sf.createConnectionForm()
	sf.createForwardingForm()
	sf.createAuthenticationForm()
	sf.createAdvancedForm()

	// Setup tab bar
	sf.updateTabBar()

	// Setup form panel
	sf.formPanel.SetBorder(true).
		SetTitle(" " + sf.titleForMode() + " ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	sf.formPanel.AddItem(sf.tabBar, 1, 0, false).
		AddItem(sf.pages, 0, 1, true)

	// Setup main container with form and help panel
	sf.mainContainer.Clear()

	// Responsive layout: hide help panel if window is too narrow
	sf.mainContainer.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		// Minimum width for showing help panel
		minWidthForHelp := 120

		// Clear and rebuild based on width
		sf.mainContainer.Clear()

		if width < minWidthForHelp || sf.helpMode == HelpModeOff {
			// Window too narrow or help is off - only show form
			sf.mainContainer.AddItem(sf.formPanel, 0, 1, true)
		} else {
			// Window wide enough - show both form and help
			sf.mainContainer.AddItem(sf.formPanel, 0, 3, true)
			sf.mainContainer.AddItem(sf.helpPanel, 0, 2, false)
			// Refresh help content to recalculate separator width on resize
			sf.updateHelp(sf.currentField)
		}

		return x, y, width, height
	})

	// Initial setup
	sf.mainContainer.AddItem(sf.formPanel, 0, 3, true)
	if sf.helpMode != HelpModeOff {
		sf.mainContainer.AddItem(sf.helpPanel, 0, 2, false)
	}

	// Create hint bar with same background as main screen's status bar
	hintBar := tview.NewTextView().SetDynamicColors(true)
	hintBar.SetBackgroundColor(tcell.Color235)
	hintBar.SetTextAlign(tview.AlignCenter)
	hintBar.SetText("[white]^H/^L[-] Navigate  • [white]^S[-] Save  • [white]Esc[-] Cancel")

	// Setup main container - header at top, hint bar at bottom
	sf.Flex.AddItem(sf.header, 2, 0, false).
		AddItem(sf.mainContainer, 0, 1, true).
		AddItem(hintBar, 1, 0, false)

	// Initialize help with first field
	sf.updateHelp("Alias")

	// Setup keyboard shortcuts
	sf.setupKeyboardShortcuts()

	// Set a draw function for the tab bar to update on each draw
	// This ensures the tab bar updates when the window is resized
	sf.tabBar.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		// Update tab bar if size changed
		sf.updateTabBar()
		// Return the original dimensions
		return x, y, width, height
	})
}

func (sf *ServerForm) titleForMode() string {
	if sf.mode == ServerFormEdit {
		return "Edit Server"
	}
	return "Add Server"
}

func (sf *ServerForm) getCurrentTabIndex() int {
	for i, tab := range sf.tabs {
		if tab == sf.currentTab {
			return i
		}
	}
	return 0
}

func (sf *ServerForm) calculateTabsWidth(useAbbrev bool) int {
	width := 0
	for i, tab := range sf.tabs {
		tabName := tab
		if useAbbrev {
			tabName = sf.tabAbbrev[tab]
		}
		width += len(tabName) + 2 // space + name + space
		if i < len(sf.tabs)-1 {
			width += 3 // " | " separator
		}
	}
	return width
}

func (sf *ServerForm) determineDisplayMode(width int) string {
	if width <= 20 { // Width unknown or too small
		return "full"
	}

	fullWidth := sf.calculateTabsWidth(false)
	if fullWidth <= width-10 {
		return "full"
	}

	abbrevWidth := sf.calculateTabsWidth(true)
	if abbrevWidth <= width-10 {
		return "abbrev"
	}

	return "scroll"
}

func (sf *ServerForm) renderTab(tab string, isCurrent bool, useAbbrev bool, index int) string {
	tabName := tab
	if useAbbrev {
		tabName = sf.tabAbbrev[tab]
	}
	regionID := fmt.Sprintf("tab_%d", index)
	if isCurrent {
		return fmt.Sprintf("[%q][black:white:b] %s [-:-:-][%q] ", regionID, tabName, "")
	}
	return fmt.Sprintf("[%q][gray::u] %s [-:-:-][%q] ", regionID, tabName, "")
}

func (sf *ServerForm) renderScrollableTabs(currentIdx, width int) string {
	var tabText string
	availableWidth := width - 8 // Reserve space for scroll indicators

	// Calculate visible count
	visibleCount := sf.calculateVisibleTabCount(availableWidth)
	if visibleCount < 2 {
		visibleCount = 2
	}

	// Add left scroll indicator
	if currentIdx > 0 {
		tabText = "[gray]◀ [-]"
	}

	// Calculate range
	start, end := sf.calculateVisibleRange(currentIdx, visibleCount, len(sf.tabs))

	// Render visible tabs
	for i := start; i < end && i < len(sf.tabs); i++ {
		tabText += sf.renderTab(sf.tabs[i], sf.tabs[i] == sf.currentTab, true, i)
		if i < end-1 && i < len(sf.tabs)-1 {
			tabText += tabSeparator
		}
	}

	// Add right scroll indicator
	if currentIdx < len(sf.tabs)-1 {
		tabText += " [gray]▶[-]"
	}

	return tabText
}

func (sf *ServerForm) calculateVisibleTabCount(availableWidth int) int {
	visibleCount := 0
	currentWidth := 0

	for i := 0; i < len(sf.tabs) && currentWidth < availableWidth; i++ {
		abbrev := sf.tabAbbrev[sf.tabs[i]]
		tabWidth := len(abbrev) + 2
		if i > 0 {
			tabWidth += 3 // separator
		}
		if currentWidth+tabWidth <= availableWidth {
			visibleCount++
			currentWidth += tabWidth
		} else {
			break
		}
	}

	return visibleCount
}

func (sf *ServerForm) calculateVisibleRange(currentIdx, visibleCount, totalTabs int) (int, int) {
	halfVisible := visibleCount / 2
	start := currentIdx - halfVisible + 1
	end := start + visibleCount

	// Adjust boundaries
	if start < 0 {
		start = 0
		end = visibleCount
	}
	if end > totalTabs {
		end = totalTabs
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	return start, end
}

func (sf *ServerForm) updateTabBar() {
	currentIdx := sf.getCurrentTabIndex()

	// Build tab text with scroll indicator if needed
	var tabText string

	// Check if we need to show scroll indicators
	x, y, width, height := sf.tabBar.GetInnerRect()
	_ = x
	_ = y
	_ = height

	displayMode := sf.determineDisplayMode(width)

	switch displayMode {
	case "scroll":
		tabText = sf.renderScrollableTabs(currentIdx, width)
	case "abbrev":
		// Show all tabs with abbreviated names
		for i, tab := range sf.tabs {
			tabText += sf.renderTab(tab, tab == sf.currentTab, true, i)
			if i < len(sf.tabs)-1 {
				tabText += tabSeparator
			}
		}
	default: // "full"
		// Show all tabs with full names
		for i, tab := range sf.tabs {
			tabText += sf.renderTab(tab, tab == sf.currentTab, false, i)
			if i < len(sf.tabs)-1 {
				tabText += tabSeparator
			}
		}
	}

	sf.tabBar.SetText(tabText)

	// Set up mouse click handler using highlight regions
	sf.tabBar.SetHighlightedFunc(func(added, removed, remaining []string) {
		if len(added) > 0 {
			// Extract tab index from region ID (format: "tab_0", "tab_1", etc)
			for _, regionID := range added {
				if len(regionID) > 4 && regionID[:4] == "tab_" {
					idx := int(regionID[4] - '0')
					if idx < len(sf.tabs) {
						sf.switchToTab(sf.tabs[idx])
					}
				}
			}
		}
	})
}

func (sf *ServerForm) switchToTab(tabName string) {
	for _, tab := range sf.tabs {
		if tab != tabName {
			continue
		}

		sf.currentTab = tabName
		sf.pages.SwitchToPage(tabName)
		sf.updateTabBar()

		// Set focus to the form in the newly selected tab
		if form, exists := sf.forms[tabName]; exists && sf.app != nil {
			sf.app.SetFocus(form)
		}
		break
	}
}

func (sf *ServerForm) nextTab() {
	for i, tab := range sf.tabs {
		if tab == sf.currentTab {
			// Loop to first tab if at the last tab
			if i == len(sf.tabs)-1 {
				sf.switchToTab(sf.tabs[0])
			} else {
				sf.switchToTab(sf.tabs[i+1])
			}
			break
		}
	}
}

func (sf *ServerForm) prevTab() {
	for i, tab := range sf.tabs {
		if tab == sf.currentTab {
			// Loop to last tab if at the first tab
			if i == 0 {
				sf.switchToTab(sf.tabs[len(sf.tabs)-1])
			} else {
				sf.switchToTab(sf.tabs[i-1])
			}
			break
		}
	}
}

// updateHelp updates the help panel with information for the given field
func (sf *ServerForm) updateHelp(fieldName string) {
	if sf.helpPanel == nil || sf.helpMode == HelpModeOff {
		return
	}

	sf.currentField = fieldName
	help := GetFieldHelp(fieldName)
	if help == nil {
		sf.helpPanel.SetText("[dim]No help available for this field[-]")
		return
	}

	var content string
	if sf.helpMode == HelpModeCompact {
		// Compact mode: single line
		example := ""
		if len(help.Examples) > 0 {
			example = help.Examples[0]
		}
		content = fmt.Sprintf("[yellow]%s:[-] %s", help.Field, escapeForTview(help.Description))
		if example != "" {
			content += fmt.Sprintf(" [dim](e.g., %s)[-]", escapeForTview(example))
		}
	} else {
		// Normal/Full mode: detailed help
		content = sf.formatDetailedHelp(help)
	}

	sf.helpPanel.SetText(content)
}

// escapeForTview escapes special characters for tview display
func escapeForTview(text string) string {
	// Use tview's own Escape function to properly escape text
	return tview.Escape(text)
}

// formatDetailedHelp formats detailed help content for a field
func (sf *ServerForm) formatDetailedHelp(help *FieldHelp) string {
	var b strings.Builder

	// Calculate separator width dynamically
	// Get the actual width of the help panel if possible
	separatorWidth := 40 // Default width
	if sf.helpPanel != nil {
		_, _, width, _ := sf.helpPanel.GetInnerRect()
		if width > 0 {
			separatorWidth = width // Fill entire width
		}
	}

	// Title with field name and separator below
	b.WriteString(fmt.Sprintf("[yellow::b]📖 %s[-::-]\n", help.Field))
	b.WriteString("[#444444]" + strings.Repeat("─", separatorWidth) + "[-]\n\n")

	// Description - needs escaping as it might contain brackets
	b.WriteString(fmt.Sprintf("%s\n\n", escapeForTview(help.Description)))

	// Syntax - needs escaping as it often contains brackets like [user@]
	if help.Syntax != "" {
		b.WriteString("[cyan]Syntax:[-] ")
		b.WriteString(fmt.Sprintf("%s\n\n", escapeForTview(help.Syntax)))
	}

	// Examples - needs escaping as they might contain special characters
	if len(help.Examples) > 0 {
		b.WriteString("[cyan]Examples:[-]\n")
		for _, ex := range help.Examples {
			b.WriteString(fmt.Sprintf("  • %s\n", escapeForTview(ex)))
		}
		b.WriteString("\n")
	}

	// Default value - already processed by formatDefaultValue, no additional escaping needed
	if help.Default != "" {
		b.WriteString(fmt.Sprintf("[dim]Default: %s[-]\n", help.Default))
	}

	// Version info - unlikely to contain brackets, but escape for safety
	if help.Since != "" {
		b.WriteString(fmt.Sprintf("[dim]Available since: %s[-]\n", escapeForTview(help.Since)))
	}

	return b.String()
}

// Note: toggleHelp and rebuildLayout were removed due to hang issues
// The help panel is now always visible on the right side

func (sf *ServerForm) setupKeyboardShortcuts() {
	// Set input capture for the main flex container
	sf.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Help panel is always visible - no toggle needed

		// Check for Ctrl key combinations with regular keys
		if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModCtrl != 0 {
			switch event.Rune() {
			case 'h', 'H', 8: // 8 is ASCII for Ctrl+H (backspace)
				// Ctrl+H: Previous tab
				sf.prevTab()
				return nil
			case 'l', 'L', 12: // 12 is ASCII for Ctrl+L (form feed)
				// Ctrl+L: Next tab
				sf.nextTab()
				return nil
			case 's', 'S', 19: // 19 is ASCII for Ctrl+S
				// Ctrl+S: Save
				sf.handleSave()
				return nil
			}
		}

		// Handle special keys
		//nolint:exhaustive // We only handle specific keys and pass through others
		switch event.Key() {
		case tcell.KeyCtrlS:
			// Ctrl+S: Save (backup handler)
			sf.handleSave()
			return nil
		case tcell.KeyEscape:
			// ESC: Cancel
			sf.handleCancel()
			return nil
		case tcell.KeyCtrlH:
			// Ctrl+H: Previous tab (backup handler)
			sf.prevTab()
			return nil
		case tcell.KeyCtrlL:
			// Ctrl+L: Next tab (backup handler)
			sf.nextTab()
			return nil
		default:
			// Pass through all other keys
		}

		return event
	})
}

// setupFormShortcuts sets up keyboard shortcuts for a form
func (sf *ServerForm) setupFormShortcuts(form *tview.Form) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Check for Ctrl key combinations
		if event.Key() == tcell.KeyRune && event.Modifiers()&tcell.ModCtrl != 0 {
			switch event.Rune() {
			case 'h', 'H', 8: // Ctrl+H: Previous tab
				sf.prevTab()
				return nil
			case 'l', 'L', 12: // Ctrl+L: Next tab
				sf.nextTab()
				return nil
			case 's', 'S', 19: // Ctrl+S: Save
				sf.handleSave()
				return nil
			}
		}

		// Handle special keys
		//nolint:exhaustive // We only handle specific keys and pass through others
		switch event.Key() {
		case tcell.KeyEscape:
			sf.handleCancel()
			return nil
		case tcell.KeyCtrlH:
			sf.prevTab()
			return nil
		case tcell.KeyCtrlL:
			sf.nextTab()
			return nil
		case tcell.KeyCtrlS:
			sf.handleSave()
			return nil
		default:
			// Pass through all other keys
		}

		return event
	})
}

// createOptionsWithDefault creates dropdown options with default value indicated
func createOptionsWithDefault(fieldName string, baseOptions []string) []string {
	defaultValue, hasDefault := sshDefaults[fieldName]
	if !hasDefault {
		return baseOptions
	}

	options := make([]string, len(baseOptions))
	for i, opt := range baseOptions {
		if opt == "" {
			options[i] = fmt.Sprintf("default [gray](%s)[-]", defaultValue)
		} else {
			options[i] = opt
		}
	}
	return options
}

// parseOptionValue extracts the actual value from an option (handles "default [gray](value)[-]" format)
func parseOptionValue(option string) string {
	// Check for colored default format
	if strings.HasPrefix(option, "default [gray](") && strings.HasSuffix(option, ")[-]") {
		return "" // Return empty string for default values
	}
	// Check for plain default format (backward compatibility)
	if strings.HasPrefix(option, "default (") && strings.HasSuffix(option, ")") {
		return "" // Return empty string for default values
	}
	return option
}

// findOptionIndex finds the index of a value in options slice
func (sf *ServerForm) findOptionIndex(options []string, value string) int {
	// Empty value should match "default [gray](...)[-]" or "default (...)" option
	if value == "" {
		for i, opt := range options {
			if strings.HasPrefix(opt, "default [gray](") || strings.HasPrefix(opt, "default (") {
				return i
			}
		}
	}

	// Look for exact match first
	for i, opt := range options {
		if strings.EqualFold(opt, value) {
			return i
		}
	}

	// Then look for options with descriptions (e.g., "none (-N)" matches "none")
	for i, opt := range options {
		// Extract the base value from options like "none (-N)"
		if spaceIdx := strings.Index(opt, " "); spaceIdx > 0 {
			baseOpt := opt[:spaceIdx]
			if strings.EqualFold(baseOpt, value) {
				return i
			}
		}
	}

	return 0 // Default to first option
}

// matchesSequence checks if all characters in pattern appear in sequence within text
func matchesSequence(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	textIdx := 0
	for _, ch := range pattern {
		found := false
		for textIdx < len(text) {
			if rune(text[textIdx]) == ch {
				found = true
				textIdx++
				break
			}
			textIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

// createSSHKeyAutocomplete creates an autocomplete function for SSH key file paths
func (sf *ServerForm) createSSHKeyAutocomplete() func(string) []string {
	return func(currentText string) []string {
		if currentText == "" {
			// Show available keys when field is empty
			availableKeys := GetAvailableSSHKeys()
			if len(availableKeys) == 0 {
				return nil
			}
			return availableKeys
		}

		// Split by comma to handle multiple keys
		keys := strings.Split(currentText, ",")
		lastKey := strings.TrimSpace(keys[len(keys)-1])

		// If the last key is empty (after a comma and space), show all available keys
		if lastKey == "" {
			availableKeys := GetAvailableSSHKeys()
			if len(availableKeys) == 0 {
				return nil
			}
			// Build suggestions with existing keys
			var suggestions []string
			prefix := ""
			if len(keys) > 1 {
				// Join all keys except the last empty one
				existingKeys := keys[:len(keys)-1]
				for i := range existingKeys {
					existingKeys[i] = strings.TrimSpace(existingKeys[i])
				}
				prefix = strings.Join(existingKeys, ", ") + ", "
			}
			for _, key := range availableKeys {
				suggestions = append(suggestions, prefix+key)
			}
			return suggestions
		}

		// Get available keys and filter based on what's being typed
		availableKeys := GetAvailableSSHKeys()
		if len(availableKeys) == 0 {
			return nil
		}

		// Convert to lowercase for case-insensitive matching
		searchTerm := strings.ToLower(lastKey)

		// Filter available keys
		var filtered []string
		prefix := ""
		if len(keys) > 1 {
			// Join all keys except the last one being typed
			existingKeys := keys[:len(keys)-1]
			for i := range existingKeys {
				existingKeys[i] = strings.TrimSpace(existingKeys[i])
			}
			prefix = strings.Join(existingKeys, ", ") + ", "
		}

		for _, key := range availableKeys {
			lowerKey := strings.ToLower(key)
			// Check if the key matches the search term
			if strings.Contains(lowerKey, searchTerm) || matchesSequence(lowerKey, searchTerm) {
				filtered = append(filtered, prefix+key)
			}
		}

		// If no matches found, return nil to allow Tab navigation
		if len(filtered) == 0 {
			return nil
		}

		return filtered
	}
}

// createKnownHostsAutocomplete creates an autocomplete function for known_hosts file paths
func (sf *ServerForm) createKnownHostsAutocomplete() func(string) []string {
	return func(currentText string) []string {
		if currentText == "" {
			// Show available known_hosts files when field is empty
			availableFiles := GetAvailableKnownHostsFiles()
			if len(availableFiles) == 0 {
				return nil
			}
			return availableFiles
		}

		// Split by space to handle multiple files
		files := strings.Split(currentText, " ")
		lastFile := strings.TrimSpace(files[len(files)-1])

		// If the last file is empty (after a space), show all available files
		if lastFile == "" {
			availableFiles := GetAvailableKnownHostsFiles()
			if len(availableFiles) == 0 {
				return nil
			}
			// Build suggestions with existing files
			var suggestions []string
			prefix := ""
			if len(files) > 1 {
				// Join all files except the last empty one
				existingFiles := files[:len(files)-1]
				for i := range existingFiles {
					existingFiles[i] = strings.TrimSpace(existingFiles[i])
				}
				prefix = strings.Join(existingFiles, " ") + " "
			}
			for _, file := range availableFiles {
				suggestions = append(suggestions, prefix+file)
			}
			return suggestions
		}

		// Get available files and filter based on what's being typed
		availableFiles := GetAvailableKnownHostsFiles()
		if len(availableFiles) == 0 {
			return nil
		}

		// Convert to lowercase for case-insensitive matching
		searchTerm := strings.ToLower(lastFile)

		// Filter available files
		var filtered []string
		prefix := ""
		if len(files) > 1 {
			// Join all files except the last one being typed
			existingFiles := files[:len(files)-1]
			for i := range existingFiles {
				existingFiles[i] = strings.TrimSpace(existingFiles[i])
			}
			prefix = strings.Join(existingFiles, " ") + " "
		}

		for _, file := range availableFiles {
			lowerFile := strings.ToLower(file)
			// Check if the file matches the search term
			if strings.Contains(lowerFile, searchTerm) || matchesSequence(lowerFile, searchTerm) {
				filtered = append(filtered, prefix+file)
			}
		}

		// If no matches found, return nil to allow Tab navigation
		if len(filtered) == 0 {
			return nil
		}

		return filtered
	}
}

// createAlgorithmAutocomplete creates an autocomplete function for algorithm input fields
func (sf *ServerForm) createAlgorithmAutocomplete(suggestions []string) func(string) []string {
	return func(currentText string) []string {
		if currentText == "" {
			// Return nil when empty to disable autocomplete, allowing Tab to navigate
			return nil
		}

		// Find the current word being typed
		words := strings.Split(currentText, ",")
		lastWord := strings.TrimSpace(words[len(words)-1])

		// If the last word is empty (after a comma), return nil to allow Tab navigation
		if lastWord == "" {
			return nil
		}

		// Handle prefix characters
		prefix := ""
		searchTerm := lastWord
		if lastWord[0] == '+' || lastWord[0] == '-' || lastWord[0] == '^' {
			prefix = string(lastWord[0])
			if len(lastWord) > 1 {
				searchTerm = lastWord[1:]
			} else {
				// Just a prefix character, show all suggestions
				searchTerm = ""
			}
		}

		// Filter suggestions - check if all characters appear in sequence
		var filtered []string
		for _, s := range suggestions {
			if searchTerm == "" || matchesSequence(strings.ToLower(s), strings.ToLower(searchTerm)) {
				// Build the complete text with the suggestion
				newWords := make([]string, len(words)-1)
				copy(newWords, words[:len(words)-1])
				newWords = append(newWords, prefix+s)
				filtered = append(filtered, strings.Join(newWords, ","))
			}
		}

		// If no matches found, return nil to allow Tab navigation
		if len(filtered) == 0 {
			return nil
		}

		return filtered
	}
}

// validateField validates a single field and updates the validation state
func (sf *ServerForm) validateField(fieldName, value string) string {
	// Get validators with context for alias checking
	originalAlias := ""
	if sf.original != nil {
		originalAlias = sf.original.Alias
	}
	fieldValidators := GetFieldValidatorsWithContext(originalAlias, sf.existingAliases)

	validator, exists := fieldValidators[fieldName]
	if !exists {
		// No validator for this field, it's valid
		sf.validation.SetError(fieldName, "")
		return ""
	}

	// Check required
	if validator.Required && strings.TrimSpace(value) == "" {
		err := fmt.Sprintf("%s is required", fieldName)
		sf.validation.SetError(fieldName, err)
		return err
	}

	// If field is empty and not required, it's valid
	if value == "" {
		sf.validation.SetError(fieldName, "")
		return ""
	}

	// Check custom validation function
	if validator.Validate != nil {
		if err := validator.Validate(value); err != nil {
			sf.validation.SetError(fieldName, err.Error())
			return err.Error()
		}
	}

	// Check regex pattern
	if validator.Pattern != nil && !validator.Pattern.MatchString(value) {
		sf.validation.SetError(fieldName, validator.Message)
		return validator.Message
	}

	// Field is valid
	sf.validation.SetError(fieldName, "")
	return ""
}

// addDropDownWithHelp adds a dropdown field with help support
func (sf *ServerForm) addDropDownWithHelp(form *tview.Form, label, fieldName string, options []string, initialOption int) {
	dropdown := tview.NewDropDown().
		SetLabel(label).
		SetOptions(options, nil).
		SetCurrentOption(initialOption)

	// Add focus handler to show help
	dropdown.SetFocusFunc(func() {
		sf.updateHelp(fieldName)
	})

	form.AddFormItem(dropdown)
}

// addInputFieldWithHelp adds a regular input field with help support
func (sf *ServerForm) addInputFieldWithHelp(form *tview.Form, label, fieldName, defaultValue string, width int, placeholder string) *tview.InputField {
	field := tview.NewInputField().
		SetLabel(label).
		SetText(defaultValue).
		SetFieldWidth(width)

	if placeholder != "" {
		field.SetPlaceholder(placeholder)
	}

	// Add focus handler to show help
	field.SetFocusFunc(func() {
		sf.updateHelp(fieldName)
	})

	form.AddFormItem(field)
	return field
}

// addValidatedInputField adds an input field with real-time validation
func (sf *ServerForm) addValidatedInputField(form *tview.Form, label, fieldName, defaultValue string, width int, placeholder string) *tview.InputField {
	// Store the original label without color tags
	originalLabel := label

	field := tview.NewInputField().
		SetLabel(label).
		SetText(defaultValue).
		SetFieldWidth(width)

	if placeholder != "" {
		field.SetPlaceholder(placeholder)
	}

	// Add change handler for real-time validation
	field.SetChangedFunc(func(text string) {
		if err := sf.validateField(fieldName, text); err != "" {
			// Show error in the label with red color
			field.SetLabel(fmt.Sprintf("[red]%s[-]", originalLabel))
		} else {
			// Clear error indication, restore original label
			field.SetLabel(originalLabel)
		}
	})

	// Add focus handler to show help
	field.SetFocusFunc(func() {
		sf.updateHelp(fieldName)
	})

	// Validate on blur (when field loses focus)
	field.SetFinishedFunc(func(key tcell.Key) {
		sf.validateField(fieldName, field.GetText())
	})

	form.AddFormItem(field)
	return field
}

// validateAllFields validates all fields in the current form
func (sf *ServerForm) validateAllFields() bool {
	// Clear all previous errors first
	sf.validation = NewValidationState()

	data := sf.getFormData()

	// Validate each field based on form data
	// Don't return early - validate all fields
	sf.validateField("Alias", data.Alias)
	sf.validateField("Host", data.Host)
	sf.validateField("Port", data.Port)
	sf.validateField("User", data.User)
	sf.validateField("Keys", data.Key)
	sf.validateField("Tags", data.Tags)

	// Connection fields
	sf.validateField("ConnectTimeout", data.ConnectTimeout)
	sf.validateField("ConnectionAttempts", data.ConnectionAttempts)
	sf.validateField("ServerAliveInterval", data.ServerAliveInterval)
	sf.validateField("ServerAliveCountMax", data.ServerAliveCountMax)
	sf.validateField("IPQoS", data.IPQoS)
	sf.validateField("BindAddress", data.BindAddress)

	// Port forwarding fields
	sf.validateField("LocalForward", data.LocalForward)
	sf.validateField("RemoteForward", data.RemoteForward)
	sf.validateField("DynamicForward", data.DynamicForward)

	// Authentication fields
	sf.validateField("NumberOfPasswordPrompts", data.NumberOfPasswordPrompts)

	// Advanced fields
	sf.validateField("CanonicalizeMaxDots", data.CanonicalizeMaxDots)
	sf.validateField("EscapeChar", data.EscapeChar)

	// Security fields
	sf.validateField("UserKnownHostsFile", data.UserKnownHostsFile)

	return !sf.validation.HasErrors()
}

// getDefaultValues returns default form values based on mode
func (sf *ServerForm) getDefaultValues() ServerFormData {
	// Determine which server data to use for defaults
	var server *domain.Server
	if sf.mode == ServerFormEdit && sf.original != nil {
		server = sf.original
	} else if sf.mode == ServerFormAdd && sf.initialData != nil {
		server = sf.initialData
	}

	// Use server values if available
	if server != nil {
		return ServerFormData{
			Alias: server.Alias,
			Host:  server.Host,
			User:  server.User,
			Port: func() string {
				if server.Port == 0 {
					return "22" // Default SSH port
				}
				return fmt.Sprint(server.Port)
			}(),
			Key:                  strings.Join(server.IdentityFiles, ", "),
			Tags:                 strings.Join(server.Tags, ", "),
			ProxyJump:            server.ProxyJump,
			ProxyCommand:         server.ProxyCommand,
			RemoteCommand:        server.RemoteCommand,
			RequestTTY:           server.RequestTTY,
			SessionType:          server.SessionType,
			ConnectTimeout:       server.ConnectTimeout,
			ConnectionAttempts:   server.ConnectionAttempts,
			BindAddress:          server.BindAddress,
			BindInterface:        server.BindInterface,
			AddressFamily:        server.AddressFamily,
			ExitOnForwardFailure: server.ExitOnForwardFailure,
			IPQoS:                server.IPQoS,
			// Hostname canonicalization
			CanonicalizeHostname:        server.CanonicalizeHostname,
			CanonicalDomains:            server.CanonicalDomains,
			CanonicalizeFallbackLocal:   server.CanonicalizeFallbackLocal,
			CanonicalizeMaxDots:         server.CanonicalizeMaxDots,
			CanonicalizePermittedCNAMEs: server.CanonicalizePermittedCNAMEs,
			GatewayPorts:                server.GatewayPorts,
			LocalForward:                strings.Join(server.LocalForward, ", "),
			RemoteForward:               strings.Join(server.RemoteForward, ", "),
			DynamicForward:              strings.Join(server.DynamicForward, ", "),
			ClearAllForwardings:         server.ClearAllForwardings,
			// Public key
			PubkeyAuthentication: server.PubkeyAuthentication,
			IdentitiesOnly:       server.IdentitiesOnly,
			// SSH Agent
			AddKeysToAgent: server.AddKeysToAgent,
			IdentityAgent:  server.IdentityAgent,
			// Password & Interactive
			PasswordAuthentication:       server.PasswordAuthentication,
			KbdInteractiveAuthentication: server.KbdInteractiveAuthentication,
			NumberOfPasswordPrompts:      server.NumberOfPasswordPrompts,
			// Advanced
			PreferredAuthentications:    server.PreferredAuthentications,
			ForwardAgent:                server.ForwardAgent,
			ForwardX11:                  server.ForwardX11,
			ForwardX11Trusted:           server.ForwardX11Trusted,
			ControlMaster:               server.ControlMaster,
			ControlPath:                 server.ControlPath,
			ControlPersist:              server.ControlPersist,
			ServerAliveInterval:         server.ServerAliveInterval,
			ServerAliveCountMax:         server.ServerAliveCountMax,
			Compression:                 server.Compression,
			TCPKeepAlive:                server.TCPKeepAlive,
			BatchMode:                   server.BatchMode,
			StrictHostKeyChecking:       server.StrictHostKeyChecking,
			UserKnownHostsFile:          server.UserKnownHostsFile,
			HostKeyAlgorithms:           server.HostKeyAlgorithms,
			PubkeyAcceptedAlgorithms:    server.PubkeyAcceptedAlgorithms,
			HostbasedAcceptedAlgorithms: server.HostbasedAcceptedAlgorithms,
			MACs:                        server.MACs,
			Ciphers:                     server.Ciphers,
			KexAlgorithms:               server.KexAlgorithms,
			VerifyHostKeyDNS:            server.VerifyHostKeyDNS,
			UpdateHostKeys:              server.UpdateHostKeys,
			HashKnownHosts:              server.HashKnownHosts,
			VisualHostKey:               server.VisualHostKey,
			LocalCommand:                server.LocalCommand,
			PermitLocalCommand:          server.PermitLocalCommand,
			EscapeChar:                  server.EscapeChar,
			SendEnv:                     strings.Join(server.SendEnv, ", "),
			SetEnv:                      strings.Join(server.SetEnv, ", "),
			LogLevel:                    server.LogLevel,
		}
	}
	// For new servers, use empty values instead of SSH defaults
	// SSH defaults will be applied by the SSH client if values are not specified
	return ServerFormData{
		Alias: "",   // Explicitly empty for new servers
		Host:  "",   // Explicitly empty for new servers
		User:  "",   // Empty for new servers (SSH will use current username)
		Port:  "22", // Keep port 22 as it's the standard SSH port
		Key:   "",   // Empty for new servers (SSH will try default keys)
		Tags:  "",

		// All other fields should be empty for new servers
		// The SSH client will use its defaults when these are not specified
		ProxyJump:            "",
		ProxyCommand:         "",
		RemoteCommand:        "",
		RequestTTY:           "",
		SessionType:          "",
		ConnectTimeout:       "",
		ConnectionAttempts:   "",
		BindAddress:          "",
		BindInterface:        "",
		AddressFamily:        "",
		ExitOnForwardFailure: "",
		IPQoS:                "",

		// Hostname canonicalization
		CanonicalizeHostname:        "",
		CanonicalDomains:            "",
		CanonicalizeFallbackLocal:   "",
		CanonicalizeMaxDots:         "",
		CanonicalizePermittedCNAMEs: "",

		// Port forwarding
		LocalForward:        "",
		RemoteForward:       "",
		DynamicForward:      "",
		ClearAllForwardings: "",
		GatewayPorts:        "",

		// Authentication
		PubkeyAuthentication:         "",
		IdentitiesOnly:               "",
		AddKeysToAgent:               "",
		IdentityAgent:                "",
		PasswordAuthentication:       "",
		KbdInteractiveAuthentication: "",
		NumberOfPasswordPrompts:      "",
		PreferredAuthentications:     "",
		PubkeyAcceptedAlgorithms:     "",
		HostbasedAcceptedAlgorithms:  "",

		// Forwarding
		ForwardAgent:      "",
		ForwardX11:        "",
		ForwardX11Trusted: "",

		// Multiplexing
		ControlMaster:  "",
		ControlPath:    "",
		ControlPersist: "",

		// Keep-alive
		ServerAliveInterval: "",
		ServerAliveCountMax: "",
		TCPKeepAlive:        "",

		// Connection
		Compression: "",
		BatchMode:   "",

		// Security
		StrictHostKeyChecking: "",
		CheckHostIP:           "",
		FingerprintHash:       "",
		UserKnownHostsFile:    "",
		HostKeyAlgorithms:     "",
		MACs:                  "",
		Ciphers:               "",
		KexAlgorithms:         "",
		VerifyHostKeyDNS:      "",
		UpdateHostKeys:        "",
		HashKnownHosts:        "",
		VisualHostKey:         "",

		// Command execution
		LocalCommand:       "",
		PermitLocalCommand: "",
		EscapeChar:         "",

		// Environment
		SendEnv: "",
		SetEnv:  "",

		// Debugging
		LogLevel: "",
	}
}

// createBasicForm creates the Basic configuration tab
func (sf *ServerForm) createBasicForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	// Add validated input fields
	sf.addValidatedInputField(form, "Alias:", "Alias", defaultValues.Alias, 20, GetFieldPlaceholder("Alias"))
	sf.addValidatedInputField(form, "Host/IP:", "Host", defaultValues.Host, 20, GetFieldPlaceholder("Host"))
	sf.addValidatedInputField(form, "User:", "User", defaultValues.User, 20, GetFieldPlaceholder("User"))
	sf.addValidatedInputField(form, "Port:", "Port", defaultValues.Port, 20, GetFieldPlaceholder("Port"))

	// Keys field with autocomplete
	keysField := sf.addValidatedInputField(form, "Keys:", "Keys", defaultValues.Key, 40, GetFieldPlaceholder("Keys"))
	keysField.SetAutocompleteFunc(sf.createSSHKeyAutocomplete())

	// Tags field
	sf.addValidatedInputField(form, "Tags:", "Tags", defaultValues.Tags, 30, GetFieldPlaceholder("Tags"))

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveButton)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Basic"] = form
	sf.pages.AddPage("Basic", form, true, true)
}

// createConnectionForm creates the Connection & Proxy tab
func (sf *ServerForm) createConnectionForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	form.AddTextView("\n[yellow]▶ Proxy & Command[-]", "", 0, 1, true, false)
	sf.addInputFieldWithHelp(form, "ProxyJump:", "ProxyJump", defaultValues.ProxyJump, 40, GetFieldPlaceholder("ProxyJump"))
	sf.addInputFieldWithHelp(form, "ProxyCommand:", "ProxyCommand", defaultValues.ProxyCommand, 40, GetFieldPlaceholder("ProxyCommand"))
	sf.addInputFieldWithHelp(form, "RemoteCommand:", "RemoteCommand", defaultValues.RemoteCommand, 40, GetFieldPlaceholder("RemoteCommand"))

	// RequestTTY dropdown
	requestTTYOptions := createOptionsWithDefault("RequestTTY", []string{"", "yes", "no", "force", "auto"})
	requestTTYIndex := sf.findOptionIndex(requestTTYOptions, defaultValues.RequestTTY)
	sf.addDropDownWithHelp(form, "RequestTTY:", "RequestTTY", requestTTYOptions, requestTTYIndex)

	// SessionType dropdown (OpenSSH 8.7+)
	sessionTypeOptions := createOptionsWithDefault("SessionType", []string{"", "none (-N)", "subsystem (-s)", "default"})
	sessionTypeIndex := sf.findOptionIndex(sessionTypeOptions, defaultValues.SessionType)
	sf.addDropDownWithHelp(form, "SessionType:", "SessionType", sessionTypeOptions, sessionTypeIndex)

	form.AddTextView("\n[yellow]▶ Connection Settings[-]", "", 0, 1, true, false)
	sf.addValidatedInputField(form, "ConnectTimeout:", "ConnectTimeout", defaultValues.ConnectTimeout, 10, GetFieldPlaceholder("ConnectTimeout"))
	sf.addValidatedInputField(form, "ConnectionAttempts:", "ConnectionAttempts", defaultValues.ConnectionAttempts, 10, GetFieldPlaceholder("ConnectionAttempts"))
	sf.addValidatedInputField(form, "IPQoS:", "IPQoS", defaultValues.IPQoS, 20, GetFieldPlaceholder("IPQoS"))

	// BatchMode dropdown (moved from Keep-Alive)
	batchModeOptions := createOptionsWithDefault("BatchMode", []string{"", "yes", "no"})
	batchModeIndex := sf.findOptionIndex(batchModeOptions, defaultValues.BatchMode)
	sf.addDropDownWithHelp(form, "BatchMode:", "BatchMode", batchModeOptions, batchModeIndex)

	form.AddTextView("\n[yellow]▶ Bind Options[-]", "", 0, 1, true, false)
	sf.addValidatedInputField(form, "BindAddress:", "BindAddress", defaultValues.BindAddress, 40, GetFieldPlaceholder("BindAddress"))

	// BindInterface dropdown with available network interfaces
	interfaceOptions := append([]string{""}, GetNetworkInterfaces()...)
	bindInterfaceIndex := sf.findOptionIndex(interfaceOptions, defaultValues.BindInterface)
	sf.addDropDownWithHelp(form, "BindInterface:", "BindInterface", interfaceOptions, bindInterfaceIndex)

	// AddressFamily dropdown
	addressFamilyOptions := createOptionsWithDefault("AddressFamily", []string{"", "any", "inet", "inet6"})
	addressFamilyIndex := sf.findOptionIndex(addressFamilyOptions, defaultValues.AddressFamily)
	sf.addDropDownWithHelp(form, "AddressFamily:", "AddressFamily", addressFamilyOptions, addressFamilyIndex)

	form.AddTextView("\n[yellow]▶ Hostname Canonicalization[-]", "", 0, 1, true, false)

	// CanonicalizeHostname dropdown
	canonicalizeOptions := createOptionsWithDefault("CanonicalizeHostname", []string{"", "yes", "no", "always"})
	canonicalizeIndex := sf.findOptionIndex(canonicalizeOptions, defaultValues.CanonicalizeHostname)
	sf.addDropDownWithHelp(form, "CanonicalizeHostname:", "CanonicalizeHostname", canonicalizeOptions, canonicalizeIndex)

	sf.addInputFieldWithHelp(form, "CanonicalDomains:", "CanonicalDomains", defaultValues.CanonicalDomains, 40, GetFieldPlaceholder("CanonicalDomains"))

	// CanonicalizeFallbackLocal dropdown
	fallbackOptions := createOptionsWithDefault("CanonicalizeFallbackLocal", []string{"", "yes", "no"})
	fallbackIndex := sf.findOptionIndex(fallbackOptions, defaultValues.CanonicalizeFallbackLocal)
	sf.addDropDownWithHelp(form, "CanonicalizeFallbackLocal:", "CanonicalizeFallbackLocal", fallbackOptions, fallbackIndex)

	sf.addValidatedInputField(form, "CanonicalizeMaxDots:", "CanonicalizeMaxDots", defaultValues.CanonicalizeMaxDots, 10, GetFieldPlaceholder("CanonicalizeMaxDots"))

	sf.addInputFieldWithHelp(form, "CanonicalizePermittedCNAMEs:", "CanonicalizePermittedCNAMEs", defaultValues.CanonicalizePermittedCNAMEs, 40, GetFieldPlaceholder("CanonicalizePermittedCNAMEs"))

	form.AddTextView("\n[yellow]▶ Keep-Alive[-]", "", 0, 1, true, false)
	sf.addValidatedInputField(form, "ServerAliveInterval:", "ServerAliveInterval", defaultValues.ServerAliveInterval, 10, GetFieldPlaceholder("ServerAliveInterval"))
	sf.addValidatedInputField(form, "ServerAliveCountMax:", "ServerAliveCountMax", defaultValues.ServerAliveCountMax, 10, GetFieldPlaceholder("ServerAliveCountMax"))

	// Compression dropdown
	compressionOptions := createOptionsWithDefault("Compression", []string{"", "yes", "no"})
	compressionIndex := sf.findOptionIndex(compressionOptions, defaultValues.Compression)
	sf.addDropDownWithHelp(form, "Compression:", "Compression", compressionOptions, compressionIndex)

	// TCPKeepAlive dropdown
	tcpKeepAliveOptions := createOptionsWithDefault("TCPKeepAlive", []string{"", "yes", "no"})
	tcpKeepAliveIndex := sf.findOptionIndex(tcpKeepAliveOptions, defaultValues.TCPKeepAlive)
	sf.addDropDownWithHelp(form, "TCPKeepAlive:", "TCPKeepAlive", tcpKeepAliveOptions, tcpKeepAliveIndex)

	form.AddTextView("\n[yellow]▶ Multiplexing[-]", "", 0, 1, true, false)
	// ControlMaster dropdown
	controlMasterOptions := createOptionsWithDefault("ControlMaster", []string{"", "yes", "no", "auto", "ask", "autoask"})
	controlMasterIndex := sf.findOptionIndex(controlMasterOptions, defaultValues.ControlMaster)
	sf.addDropDownWithHelp(form, "ControlMaster:", "ControlMaster", controlMasterOptions, controlMasterIndex)
	sf.addInputFieldWithHelp(form, "ControlPath:", "ControlPath", defaultValues.ControlPath, 40, GetFieldPlaceholder("ControlPath"))
	sf.addInputFieldWithHelp(form, "ControlPersist:", "ControlPersist", defaultValues.ControlPersist, 20, GetFieldPlaceholder("ControlPersist"))

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveButton)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Connection"] = form
	sf.pages.AddPage("Connection", form, true, false)
}

// createForwardingForm creates the Port Forwarding tab
func (sf *ServerForm) createForwardingForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	form.AddTextView("\n[yellow]▶ Port Forwarding[-]", "", 0, 1, true, false)
	sf.addValidatedInputField(form, "LocalForward:", "LocalForward", defaultValues.LocalForward, 40, GetFieldPlaceholder("LocalForward"))
	sf.addValidatedInputField(form, "RemoteForward:", "RemoteForward", defaultValues.RemoteForward, 40, GetFieldPlaceholder("RemoteForward"))
	sf.addValidatedInputField(form, "DynamicForward:", "DynamicForward", defaultValues.DynamicForward, 40, GetFieldPlaceholder("DynamicForward"))

	// ClearAllForwardings dropdown
	clearAllForwardingsOptions := createOptionsWithDefault("ClearAllForwardings", []string{"", "yes", "no"})
	clearAllForwardingsIndex := sf.findOptionIndex(clearAllForwardingsOptions, defaultValues.ClearAllForwardings)
	sf.addDropDownWithHelp(form, "ClearAllForwardings:", "ClearAllForwardings", clearAllForwardingsOptions, clearAllForwardingsIndex)

	// ExitOnForwardFailure dropdown
	exitOnForwardFailureOptions := createOptionsWithDefault("ExitOnForwardFailure", []string{"", "yes", "no"})
	exitOnForwardFailureIndex := sf.findOptionIndex(exitOnForwardFailureOptions, defaultValues.ExitOnForwardFailure)
	sf.addDropDownWithHelp(form, "ExitOnForwardFailure:", "ExitOnForwardFailure", exitOnForwardFailureOptions, exitOnForwardFailureIndex)

	// GatewayPorts dropdown
	gatewayPortsOptions := createOptionsWithDefault("GatewayPorts", []string{"", "yes", "no", "clientspecified"})
	gatewayPortsIndex := sf.findOptionIndex(gatewayPortsOptions, defaultValues.GatewayPorts)
	sf.addDropDownWithHelp(form, "GatewayPorts:", "GatewayPorts", gatewayPortsOptions, gatewayPortsIndex)

	form.AddTextView("\n[yellow]▶ Agent & X11 Forwarding[-]", "", 0, 1, true, false)

	// ForwardAgent dropdown
	forwardAgentOptions := createOptionsWithDefault("ForwardAgent", []string{"", "yes", "no"})
	forwardAgentIndex := sf.findOptionIndex(forwardAgentOptions, defaultValues.ForwardAgent)
	sf.addDropDownWithHelp(form, "ForwardAgent:", "ForwardAgent", forwardAgentOptions, forwardAgentIndex)

	// ForwardX11 dropdown
	forwardX11Options := createOptionsWithDefault("ForwardX11", []string{"", "yes", "no"})
	forwardX11Index := sf.findOptionIndex(forwardX11Options, defaultValues.ForwardX11)
	sf.addDropDownWithHelp(form, "ForwardX11:", "ForwardX11", forwardX11Options, forwardX11Index)

	// ForwardX11Trusted dropdown
	forwardX11TrustedOptions := createOptionsWithDefault("ForwardX11Trusted", []string{"", "yes", "no"})
	forwardX11TrustedIndex := sf.findOptionIndex(forwardX11TrustedOptions, defaultValues.ForwardX11Trusted)
	sf.addDropDownWithHelp(form, "ForwardX11Trusted:", "ForwardX11Trusted", forwardX11TrustedOptions, forwardX11TrustedIndex)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveButton)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Forwarding"] = form
	sf.pages.AddPage("Forwarding", form, true, false)
}

// Algorithm suggestions for autocomplete
var (
	pubkeyAlgorithms = []string{
		"ssh-ed25519", "ssh-ed25519-cert-v01@openssh.com",
		"sk-ssh-ed25519@openssh.com", "sk-ssh-ed25519-cert-v01@openssh.com",
		"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
		"ecdsa-sha2-nistp256-cert-v01@openssh.com",
		"ecdsa-sha2-nistp384-cert-v01@openssh.com",
		"ecdsa-sha2-nistp521-cert-v01@openssh.com",
		"sk-ecdsa-sha2-nistp256@openssh.com",
		"sk-ecdsa-sha2-nistp256-cert-v01@openssh.com",
		"rsa-sha2-512", "rsa-sha2-256",
		"rsa-sha2-512-cert-v01@openssh.com",
		"rsa-sha2-256-cert-v01@openssh.com",
		"ssh-rsa", "ssh-rsa-cert-v01@openssh.com",
		"ssh-dss", "ssh-dss-cert-v01@openssh.com",
	}

	cipherAlgorithms = []string{
		"aes128-ctr", "aes192-ctr", "aes256-ctr",
		"aes128-gcm@openssh.com", "aes256-gcm@openssh.com",
		"chacha20-poly1305@openssh.com",
		"aes128-cbc", "aes192-cbc", "aes256-cbc", "3des-cbc",
	}

	macAlgorithms = []string{
		"hmac-sha2-256", "hmac-sha2-512",
		"hmac-sha2-256-etm@openssh.com", "hmac-sha2-512-etm@openssh.com",
		"umac-64@openssh.com", "umac-128@openssh.com",
		"umac-64-etm@openssh.com", "umac-128-etm@openssh.com",
		"hmac-sha1", "hmac-sha1-96",
		"hmac-sha1-etm@openssh.com", "hmac-sha1-96-etm@openssh.com",
		"hmac-md5", "hmac-md5-96",
		"hmac-md5-etm@openssh.com", "hmac-md5-96-etm@openssh.com",
	}

	kexAlgorithms = []string{
		"curve25519-sha256", "curve25519-sha256@libssh.org",
		"ecdh-sha2-nistp256", "ecdh-sha2-nistp384", "ecdh-sha2-nistp521",
		"diffie-hellman-group-exchange-sha256",
		"diffie-hellman-group16-sha512", "diffie-hellman-group18-sha512",
		"diffie-hellman-group14-sha256", "diffie-hellman-group14-sha1",
		"diffie-hellman-group-exchange-sha1",
		"diffie-hellman-group1-sha1",
	}

	hostKeyAlgorithms = []string{
		"ssh-ed25519", "ssh-ed25519-cert-v01@openssh.com",
		"sk-ssh-ed25519@openssh.com", "sk-ssh-ed25519-cert-v01@openssh.com",
		"ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521",
		"ecdsa-sha2-nistp256-cert-v01@openssh.com",
		"ecdsa-sha2-nistp384-cert-v01@openssh.com",
		"ecdsa-sha2-nistp521-cert-v01@openssh.com",
		"rsa-sha2-512", "rsa-sha2-256",
		"rsa-sha2-512-cert-v01@openssh.com",
		"rsa-sha2-256-cert-v01@openssh.com",
		"ssh-rsa", "ssh-rsa-cert-v01@openssh.com",
		"ssh-dss", "ssh-dss-cert-v01@openssh.com",
	}
)

// createAuthenticationForm creates the Authentication tab
func (sf *ServerForm) createAuthenticationForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	// Most common: Public key authentication
	form.AddTextView("\n[yellow]▶ Public Key Authentication[-]", "", 0, 1, true, false)

	// PubkeyAuthentication dropdown
	pubkeyOptions := createOptionsWithDefault("PubkeyAuthentication", []string{"", "yes", "no"})
	pubkeyIndex := sf.findOptionIndex(pubkeyOptions, defaultValues.PubkeyAuthentication)
	sf.addDropDownWithHelp(form, "PubkeyAuthentication:", "PubkeyAuthentication", pubkeyOptions, pubkeyIndex)

	// IdentitiesOnly dropdown - controls whether to use only specified identity files
	identitiesOnlyOptions := createOptionsWithDefault("IdentitiesOnly", []string{"", "yes", "no"})
	identitiesOnlyIndex := sf.findOptionIndex(identitiesOnlyOptions, defaultValues.IdentitiesOnly)
	sf.addDropDownWithHelp(form, "IdentitiesOnly:", "IdentitiesOnly", identitiesOnlyOptions, identitiesOnlyIndex)

	// SSH Agent settings
	form.AddTextView("\n[yellow]▶ SSH Agent[-]", "", 0, 1, true, false)

	// AddKeysToAgent dropdown
	addKeysOptions := createOptionsWithDefault("AddKeysToAgent", []string{"", "yes", "no", "ask", "confirm"})
	addKeysIndex := sf.findOptionIndex(addKeysOptions, defaultValues.AddKeysToAgent)
	sf.addDropDownWithHelp(form, "AddKeysToAgent:", "AddKeysToAgent", addKeysOptions, addKeysIndex)

	sf.addInputFieldWithHelp(form, "IdentityAgent:", "IdentityAgent", defaultValues.IdentityAgent, 40, GetFieldPlaceholder("IdentityAgent"))

	// Password/Interactive authentication
	form.AddTextView("\n[yellow]▶ Password & Interactive[-]", "", 0, 1, true, false)

	// PasswordAuthentication dropdown
	passwordOptions := createOptionsWithDefault("PasswordAuthentication", []string{"", "yes", "no"})
	passwordIndex := sf.findOptionIndex(passwordOptions, defaultValues.PasswordAuthentication)
	sf.addDropDownWithHelp(form, "PasswordAuthentication:", "PasswordAuthentication", passwordOptions, passwordIndex)

	// KbdInteractiveAuthentication dropdown
	kbdInteractiveOptions := createOptionsWithDefault("KbdInteractiveAuthentication", []string{"", "yes", "no"})
	kbdInteractiveIndex := sf.findOptionIndex(kbdInteractiveOptions, defaultValues.KbdInteractiveAuthentication)
	sf.addDropDownWithHelp(form, "KbdInteractiveAuthentication:", "KbdInteractiveAuthentication", kbdInteractiveOptions, kbdInteractiveIndex)

	// NumberOfPasswordPrompts field
	sf.addValidatedInputField(form, "NumberOfPasswordPrompts:", "NumberOfPasswordPrompts", defaultValues.NumberOfPasswordPrompts, 10, GetFieldPlaceholder("NumberOfPasswordPrompts"))

	// Advanced: Authentication order preference
	form.AddTextView("\n[yellow]▶ Advanced[-]", "", 0, 1, true, false)

	sf.addInputFieldWithHelp(form, "PreferredAuthentications:", "PreferredAuthentications", defaultValues.PreferredAuthentications, 40, GetFieldPlaceholder("PreferredAuthentications"))

	// PubkeyAcceptedAlgorithms with autocomplete support (moved from Advanced/Cryptography)
	pubkeyAlgField := sf.addInputFieldWithHelp(form, "PubkeyAcceptedAlgorithms:", "PubkeyAcceptedAlgorithms", defaultValues.PubkeyAcceptedAlgorithms, 40, GetFieldPlaceholder("PubkeyAcceptedAlgorithms"))
	pubkeyAlgField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(pubkeyAlgorithms))

	// HostbasedAcceptedAlgorithms with autocomplete support (moved from Advanced/Cryptography)
	hostbasedAlgField := sf.addInputFieldWithHelp(form, "HostbasedAcceptedAlgorithms:", "HostbasedAcceptedAlgorithms", defaultValues.HostbasedAcceptedAlgorithms, 40, GetFieldPlaceholder("HostbasedAcceptedAlgorithms"))
	hostbasedAlgField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(pubkeyAlgorithms))

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveButton)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Authentication"] = form
	sf.pages.AddPage("Authentication", form, true, false)
}

// createAdvancedForm creates the Advanced settings tab
func (sf *ServerForm) createAdvancedForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	form.AddTextView("\n[yellow]▶ Security[-]", "", 0, 1, true, false)

	// StrictHostKeyChecking dropdown
	strictHostKeyOptions := createOptionsWithDefault("StrictHostKeyChecking", []string{"", "yes", "no", "ask", "accept-new"})
	strictHostKeyIndex := sf.findOptionIndex(strictHostKeyOptions, defaultValues.StrictHostKeyChecking)
	sf.addDropDownWithHelp(form, "StrictHostKeyChecking:", "StrictHostKeyChecking", strictHostKeyOptions, strictHostKeyIndex)

	// CheckHostIP dropdown
	checkHostIPOptions := createOptionsWithDefault("CheckHostIP", []string{"", "yes", "no"})
	checkHostIPIndex := sf.findOptionIndex(checkHostIPOptions, defaultValues.CheckHostIP)
	sf.addDropDownWithHelp(form, "CheckHostIP:", "CheckHostIP", checkHostIPOptions, checkHostIPIndex)

	// FingerprintHash dropdown
	fingerprintHashOptions := createOptionsWithDefault("FingerprintHash", []string{"", "md5", "sha256"})
	fingerprintHashIndex := sf.findOptionIndex(fingerprintHashOptions, defaultValues.FingerprintHash)
	sf.addDropDownWithHelp(form, "FingerprintHash:", "FingerprintHash", fingerprintHashOptions, fingerprintHashIndex)

	// VerifyHostKeyDNS dropdown
	verifyHostKeyDNSOptions := createOptionsWithDefault("VerifyHostKeyDNS", []string{"", "yes", "no", "ask"})
	verifyHostKeyDNSIndex := sf.findOptionIndex(verifyHostKeyDNSOptions, defaultValues.VerifyHostKeyDNS)
	sf.addDropDownWithHelp(form, "VerifyHostKeyDNS:", "VerifyHostKeyDNS", verifyHostKeyDNSOptions, verifyHostKeyDNSIndex)

	// UpdateHostKeys dropdown
	updateHostKeysOptions := createOptionsWithDefault("UpdateHostKeys", []string{"", "yes", "no", "ask"})
	updateHostKeysIndex := sf.findOptionIndex(updateHostKeysOptions, defaultValues.UpdateHostKeys)
	sf.addDropDownWithHelp(form, "UpdateHostKeys:", "UpdateHostKeys", updateHostKeysOptions, updateHostKeysIndex)

	// HashKnownHosts dropdown
	hashKnownHostsOptions := createOptionsWithDefault("HashKnownHosts", []string{"", "yes", "no"})
	hashKnownHostsIndex := sf.findOptionIndex(hashKnownHostsOptions, defaultValues.HashKnownHosts)
	sf.addDropDownWithHelp(form, "HashKnownHosts:", "HashKnownHosts", hashKnownHostsOptions, hashKnownHostsIndex)

	// VisualHostKey dropdown
	visualHostKeyOptions := createOptionsWithDefault("VisualHostKey", []string{"", "yes", "no"})
	visualHostKeyIndex := sf.findOptionIndex(visualHostKeyOptions, defaultValues.VisualHostKey)
	sf.addDropDownWithHelp(form, "VisualHostKey:", "VisualHostKey", visualHostKeyOptions, visualHostKeyIndex)

	// UserKnownHostsFile field with autocomplete and validation
	knownHostsField := sf.addValidatedInputField(form, "UserKnownHostsFile:", "UserKnownHostsFile", defaultValues.UserKnownHostsFile, 40, GetFieldPlaceholder("UserKnownHostsFile"))
	knownHostsField.SetAutocompleteFunc(sf.createKnownHostsAutocomplete())

	form.AddTextView("\n[yellow]▶ Cryptography[-]", "", 0, 1, true, false)

	// Ciphers with autocomplete support
	ciphersField := sf.addInputFieldWithHelp(form, "Ciphers:", "Ciphers", defaultValues.Ciphers, 40, GetFieldPlaceholder("Ciphers"))
	ciphersField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(cipherAlgorithms))

	// MACs with autocomplete support
	macsField := sf.addInputFieldWithHelp(form, "MACs:", "MACs", defaultValues.MACs, 40, GetFieldPlaceholder("MACs"))
	macsField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(macAlgorithms))

	// KexAlgorithms with autocomplete support
	kexField := sf.addInputFieldWithHelp(form, "KexAlgorithms:", "KexAlgorithms", defaultValues.KexAlgorithms, 40, GetFieldPlaceholder("KexAlgorithms"))
	kexField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(kexAlgorithms))

	// HostKeyAlgorithms with autocomplete support
	hostKeyField := sf.addInputFieldWithHelp(form, "HostKeyAlgorithms:", "HostKeyAlgorithms", defaultValues.HostKeyAlgorithms, 40, GetFieldPlaceholder("HostKeyAlgorithms"))
	hostKeyField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(hostKeyAlgorithms))

	form.AddTextView("\n[yellow]▶ Command Execution[-]", "", 0, 1, true, false)
	sf.addInputFieldWithHelp(form, "LocalCommand:", "LocalCommand", defaultValues.LocalCommand, 40, GetFieldPlaceholder("LocalCommand"))

	// PermitLocalCommand dropdown
	permitLocalCommandOptions := createOptionsWithDefault("PermitLocalCommand", []string{"", "yes", "no"})
	permitLocalCommandIndex := sf.findOptionIndex(permitLocalCommandOptions, defaultValues.PermitLocalCommand)
	sf.addDropDownWithHelp(form, "PermitLocalCommand:", "PermitLocalCommand", permitLocalCommandOptions, permitLocalCommandIndex)

	// EscapeChar input field
	sf.addValidatedInputField(form, "EscapeChar:", "EscapeChar", defaultValues.EscapeChar, 10, GetFieldPlaceholder("EscapeChar"))

	form.AddTextView("\n[yellow]▶ Environment[-]", "", 0, 1, true, false)
	sf.addInputFieldWithHelp(form, "SendEnv:", "SendEnv", defaultValues.SendEnv, 40, GetFieldPlaceholder("SendEnv"))
	sf.addInputFieldWithHelp(form, "SetEnv:", "SetEnv", defaultValues.SetEnv, 40, GetFieldPlaceholder("SetEnv"))

	form.AddTextView("\n[yellow]▶ Debugging[-]", "", 0, 1, true, false)

	// LogLevel dropdown
	logLevelOptions := createOptionsWithDefault("LogLevel", []string{"", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"})
	logLevelIndex := sf.findOptionIndex(logLevelOptions, defaultValues.LogLevel)
	sf.addDropDownWithHelp(form, "LogLevel:", "LogLevel", logLevelOptions, logLevelIndex)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveButton)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Advanced"] = form
	sf.pages.AddPage("Advanced", form, true, false)
}

type ServerFormData struct {
	Alias string
	Host  string
	User  string
	Port  string
	Key   string
	Tags  string

	// Connection and proxy settings
	ProxyJump            string
	ProxyCommand         string
	RemoteCommand        string
	RequestTTY           string
	SessionType          string
	ConnectTimeout       string
	ConnectionAttempts   string
	BindAddress          string
	BindInterface        string
	AddressFamily        string
	ExitOnForwardFailure string
	IPQoS                string
	// Hostname canonicalization
	CanonicalizeHostname        string
	CanonicalDomains            string
	CanonicalizeFallbackLocal   string
	CanonicalizeMaxDots         string
	CanonicalizePermittedCNAMEs string

	// Port forwarding
	LocalForward        string
	RemoteForward       string
	DynamicForward      string
	ClearAllForwardings string
	GatewayPorts        string

	// Authentication and key management
	// Public key
	PubkeyAuthentication string
	IdentitiesOnly       string
	// SSH Agent
	AddKeysToAgent string
	IdentityAgent  string
	// Password & Interactive
	PasswordAuthentication       string
	KbdInteractiveAuthentication string
	NumberOfPasswordPrompts      string
	// Advanced
	PreferredAuthentications string

	// Agent and X11 forwarding
	ForwardAgent      string
	ForwardX11        string
	ForwardX11Trusted string

	// Connection multiplexing
	ControlMaster  string
	ControlPath    string
	ControlPersist string

	// Connection reliability
	ServerAliveInterval string
	ServerAliveCountMax string
	Compression         string
	TCPKeepAlive        string
	BatchMode           string

	// Security settings
	StrictHostKeyChecking       string
	CheckHostIP                 string
	FingerprintHash             string
	UserKnownHostsFile          string
	HostKeyAlgorithms           string
	PubkeyAcceptedAlgorithms    string
	HostbasedAcceptedAlgorithms string
	MACs                        string
	Ciphers                     string
	KexAlgorithms               string
	VerifyHostKeyDNS            string
	UpdateHostKeys              string
	HashKnownHosts              string
	VisualHostKey               string

	// Command execution
	LocalCommand       string
	PermitLocalCommand string
	EscapeChar         string

	// Environment settings
	SendEnv string
	SetEnv  string

	// Debugging settings
	LogLevel string
}

// stripColorTags removes tview color tags from a string
// e.g., "[red]Port:[-]" becomes "Port:"
func (sf *ServerForm) getFormData() ServerFormData {
	// Helper function to get text from InputField across all forms
	getFieldText := func(fieldName string) string {
		for _, form := range sf.forms {
			for i := 0; i < form.GetFormItemCount(); i++ {
				if field, ok := form.GetFormItem(i).(*tview.InputField); ok {
					label := strings.TrimSpace(field.GetLabel())
					// Strip color tags from label for comparison
					// Labels can be: "Port:", "[red]Port:[-]", "[green]Port:[-]"
					cleanLabel := stripColorTags(label)
					if strings.HasPrefix(cleanLabel, fieldName) {
						return strings.TrimSpace(field.GetText())
					}
				}
			}
		}
		return ""
	}

	// Helper function to get selected option from DropDown across all forms
	getDropdownValue := func(fieldName string) string {
		for _, form := range sf.forms {
			for i := 0; i < form.GetFormItemCount(); i++ {
				if dropdown, ok := form.GetFormItem(i).(*tview.DropDown); ok {
					label := strings.TrimSpace(dropdown.GetLabel())
					// Strip color tags from label for comparison
					cleanLabel := stripColorTags(label)
					if strings.HasPrefix(cleanLabel, fieldName) {
						_, text := dropdown.GetCurrentOption()
						// Parse the option value to handle "default (value)" format
						return parseOptionValue(text)
					}
				}
			}
		}
		return ""
	}

	return ServerFormData{
		Alias: getFieldText("Alias:"),
		Host:  getFieldText("Host/IP:"),
		User:  getFieldText("User:"),
		Port:  getFieldText("Port:"),
		Key:   getFieldText("Keys:"),
		Tags:  getFieldText("Tags:"),
		// Connection and proxy settings
		ProxyJump:            getFieldText("ProxyJump:"),
		ProxyCommand:         getFieldText("ProxyCommand:"),
		RemoteCommand:        getFieldText("RemoteCommand:"),
		RequestTTY:           getDropdownValue("RequestTTY:"),
		SessionType:          sf.parseSessionType(getDropdownValue("SessionType:")),
		ConnectTimeout:       getFieldText("ConnectTimeout:"),
		ConnectionAttempts:   getFieldText("ConnectionAttempts:"),
		BindAddress:          getFieldText("BindAddress:"),
		BindInterface:        getDropdownValue("BindInterface:"),
		AddressFamily:        getDropdownValue("AddressFamily:"),
		ExitOnForwardFailure: getDropdownValue("ExitOnForwardFailure:"),
		// Port forwarding
		LocalForward:        getFieldText("LocalForward:"),
		RemoteForward:       getFieldText("RemoteForward:"),
		DynamicForward:      getFieldText("DynamicForward:"),
		ClearAllForwardings: getDropdownValue("ClearAllForwardings:"),
		// Authentication and key management
		// Public key
		PubkeyAuthentication: getDropdownValue("PubkeyAuthentication:"),
		IdentitiesOnly:       getDropdownValue("IdentitiesOnly:"),
		// SSH Agent
		AddKeysToAgent: getDropdownValue("AddKeysToAgent:"),
		IdentityAgent:  getFieldText("IdentityAgent:"),
		// Password & Interactive
		PasswordAuthentication:       getDropdownValue("PasswordAuthentication:"),
		KbdInteractiveAuthentication: getDropdownValue("KbdInteractiveAuthentication:"),
		NumberOfPasswordPrompts:      getFieldText("NumberOfPasswordPrompts:"),
		// Advanced
		PreferredAuthentications: getFieldText("PreferredAuthentications:"),
		// Agent and X11 forwarding
		ForwardAgent:      getDropdownValue("ForwardAgent:"),
		ForwardX11:        getDropdownValue("ForwardX11:"),
		ForwardX11Trusted: getDropdownValue("ForwardX11Trusted:"),
		// Connection multiplexing
		ControlMaster:  getDropdownValue("ControlMaster:"),
		ControlPath:    getFieldText("ControlPath:"),
		ControlPersist: getFieldText("ControlPersist:"),
		// Connection reliability settings
		ServerAliveInterval: getFieldText("ServerAliveInterval:"),
		ServerAliveCountMax: getFieldText("ServerAliveCountMax:"),
		Compression:         getDropdownValue("Compression:"),
		TCPKeepAlive:        getDropdownValue("TCPKeepAlive:"),
		BatchMode:           getDropdownValue("BatchMode:"),
		// Security settings
		StrictHostKeyChecking:    getDropdownValue("StrictHostKeyChecking:"),
		UserKnownHostsFile:       getFieldText("UserKnownHostsFile:"),
		HostKeyAlgorithms:        getFieldText("HostKeyAlgorithms:"),
		PubkeyAcceptedAlgorithms: getFieldText("PubkeyAcceptedAlgorithms:"),
		MACs:                     getFieldText("MACs:"),
		Ciphers:                  getFieldText("Ciphers:"),
		KexAlgorithms:            getFieldText("KexAlgorithms:"),
		VerifyHostKeyDNS:         getDropdownValue("VerifyHostKeyDNS:"),
		UpdateHostKeys:           getDropdownValue("UpdateHostKeys:"),
		HashKnownHosts:           getDropdownValue("HashKnownHosts:"),
		VisualHostKey:            getDropdownValue("VisualHostKey:"),
		// Command execution
		LocalCommand:       getFieldText("LocalCommand:"),
		PermitLocalCommand: getDropdownValue("PermitLocalCommand:"),
		EscapeChar:         getFieldText("EscapeChar:"),
		// Environment settings
		SendEnv: getFieldText("SendEnv:"),
		SetEnv:  getFieldText("SetEnv:"),
		// Debugging settings
		LogLevel: getDropdownValue("LogLevel:"),
	}
}

// parseSessionType converts dropdown display value to actual value
func (sf *ServerForm) parseSessionType(value string) string {
	// First handle the default value format
	if strings.HasPrefix(value, "default (") && strings.HasSuffix(value, ")") {
		return "" // Return empty string for default values
	}

	// Then handle specific display values
	switch value {
	case "none (-N)":
		return sessionTypeNone
	case "subsystem (-s)":
		return sessionTypeSubsystem
	case "default":
		return sessionTypeDefault
	default:
		return value
	}
}

// handleSaveButton is a wrapper for button callback (no return value)
func (sf *ServerForm) handleSaveButton() {
	sf.handleSave()
}

// handleSave validates and saves the form, returns true if successful
func (sf *ServerForm) handleSave() bool {
	// First validate all fields with the new validation system
	if !sf.validateAllFields() {
		// Show validation errors
		if sf.app != nil {
			errors := sf.validation.GetAllErrors()
			if len(errors) > 0 {
				// Limit the number of errors to display to prevent overflow
				maxErrorsToShow := 5
				truncated := false
				if len(errors) > maxErrorsToShow {
					errors = errors[:maxErrorsToShow]
					truncated = true
				}

				// Build error message
				errorMsg := fmt.Sprintf("Validation failed (%d error%s):\n\n",
					sf.validation.GetErrorCount(),
					func() string {
						if sf.validation.GetErrorCount() == 1 {
							return ""
						}
						return "s"
					}())

				for i, err := range errors {
					errorMsg += fmt.Sprintf("%d. %s\n", i+1, err)
				}

				if truncated {
					errorMsg += fmt.Sprintf("\n... and %d more error%s",
						sf.validation.GetErrorCount()-maxErrorsToShow,
						func() string {
							if sf.validation.GetErrorCount()-maxErrorsToShow == 1 {
								return ""
							}
							return "s"
						}())
				}

				// Use tview's built-in Modal
				modal := tview.NewModal().
					SetText(errorMsg).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						sf.app.SetRoot(sf.Flex, true)
					})

				sf.app.SetRoot(modal, true)
			}
		}
		return false // Validation failed
	}

	data := sf.getFormData()

	// Reset title and border (validation already done above)
	sf.formPanel.SetTitle(" " + sf.titleForMode() + " ")
	sf.formPanel.SetBorderColor(tcell.Color238)

	server := sf.dataToServer(data)
	if sf.onSave != nil {
		sf.onSave(server, sf.original)
	}
	return true // Save successful
}

func (sf *ServerForm) handleCancel() {
	// Check if there are unsaved changes
	if sf.hasUnsavedChanges() {
		// If app reference is available, show confirmation dialog
		if sf.app != nil {
			modal := tview.NewModal().
				SetText("You have unsaved changes. Are you sure you want to exit?").
				AddButtons([]string{"[yellow]S[-]ave", "[yellow]D[-]iscard", "[yellow]C[-]ancel"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonIndex {
					case 0: // Save
						// Try to save, if successful it will exit
						if sf.handleSave() {
							// Save successful, modal will be replaced by onSave callback
						} else {
							// Validation failed, return to form
							sf.app.SetRoot(sf.Flex, true)
						}
					case 1: // Discard
						if sf.onCancel != nil {
							sf.onCancel()
						}
					case 2: // Cancel
						// Restore the form view
						sf.app.SetRoot(sf.Flex, true)
					}
				})

			// Set up keyboard shortcuts for the modal
			modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Rune() {
				case 's', 'S':
					if sf.handleSave() {
						// Save successful
					} else {
						// Validation failed, return to form
						sf.app.SetRoot(sf.Flex, true)
					}
					return nil
				case 'd', 'D':
					if sf.onCancel != nil {
						sf.onCancel()
					}
					return nil
				case 'c', 'C':
					sf.app.SetRoot(sf.Flex, true)
					return nil
				}
				return event
			})

			// Show modal
			sf.app.SetRoot(modal, true)
		} else if sf.onCancel != nil {
			// No app reference, fallback to direct cancel (shouldn't happen in normal use)
			sf.onCancel()
		}
	} else {
		// No unsaved changes, just exit
		if sf.onCancel != nil {
			sf.onCancel()
		}
	}
}

// hasUnsavedChanges checks if current form data differs from original
func (sf *ServerForm) hasUnsavedChanges() bool {
	// If we have original data to compare against (Edit mode or Add with paste)
	if sf.original != nil {
		currentData := sf.getFormData()
		currentServer := sf.dataToServer(currentData)

		// Use DeepEqual for simple comparison first
		if reflect.DeepEqual(currentServer, *sf.original) {
			return false
		}

		// If DeepEqual says they're different, use our custom comparison
		// that handles nil vs empty slice and other normalization
		return sf.serversDiffer(currentServer, *sf.original)
	}

	// For new servers without original (regular Add), check if any data has been entered
	if sf.mode == ServerFormAdd {
		data := sf.getFormData()
		return data.Alias != "" || data.Host != "" || data.User != ""
	}

	return false
}

// serversDiffer compares two servers for differences using reflection
func (sf *ServerForm) serversDiffer(a, b domain.Server) bool {
	// Use reflection to compare all fields
	valA := reflect.ValueOf(a)
	valB := reflect.ValueOf(b)
	typeA := valA.Type()

	// Fields to skip during comparison (lazyssh metadata fields)
	skipFields := map[string]bool{
		"Aliases":  true, // Computed field
		"LastSeen": true, // Metadata field
		"PinnedAt": true, // Metadata field
		"SSHCount": true, // Metadata field
	}

	// Iterate through all fields
	for i := 0; i < valA.NumField(); i++ {
		fieldA := valA.Field(i)
		fieldB := valB.Field(i)
		fieldName := typeA.Field(i).Name

		// Skip unexported fields and metadata fields
		if !fieldA.CanInterface() || skipFields[fieldName] {
			continue
		}

		// Compare based on field type
		differs := false
		switch fieldA.Kind() {
		case reflect.String:
			if fieldA.String() != fieldB.String() {
				differs = true
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if fieldA.Int() != fieldB.Int() {
				differs = true
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if fieldA.Uint() != fieldB.Uint() {
				differs = true
			}
		case reflect.Slice:
			if !sf.slicesEqual(fieldA, fieldB) {
				differs = true
			}
		case reflect.Bool:
			if fieldA.Bool() != fieldB.Bool() {
				differs = true
			}
		case reflect.Float32, reflect.Float64:
			if fieldA.Float() != fieldB.Float() {
				differs = true
			}
		case reflect.Complex64, reflect.Complex128:
			if fieldA.Complex() != fieldB.Complex() {
				differs = true
			}
		case reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer, reflect.Invalid:
			// For these types, use reflect.DeepEqual
			if !reflect.DeepEqual(fieldA.Interface(), fieldB.Interface()) {
				differs = true
			}
		}

		if differs {
			return true
		}
	}

	return false
}

// slicesEqual compares two reflect.Value slices for equality
func (sf *ServerForm) slicesEqual(a, b reflect.Value) bool {
	// Handle nil slices - treat nil and empty slice as equal
	if a.IsNil() && b.IsNil() {
		return true
	}
	if a.IsNil() && b.Len() == 0 {
		return true
	}
	if b.IsNil() && a.Len() == 0 {
		return true
	}

	if a.Len() != b.Len() {
		return false
	}

	for i := 0; i < a.Len(); i++ {
		// For string slices
		if a.Index(i).Kind() == reflect.String {
			if a.Index(i).String() != b.Index(i).String() {
				return false
			}
		} else {
			// For other types, use DeepEqual
			if !reflect.DeepEqual(a.Index(i).Interface(), b.Index(i).Interface()) {
				return false
			}
		}
	}

	return true
}

func (sf *ServerForm) dataToServer(data ServerFormData) domain.Server {
	port := 22
	if data.Port != "" {
		if n, err := strconv.Atoi(data.Port); err == nil && n > 0 {
			port = n
		}
	}

	// Use nil for empty slices to match original state
	var tags []string
	if data.Tags != "" {
		for _, t := range strings.Split(data.Tags, ",") {
			if s := strings.TrimSpace(t); s != "" {
				tags = append(tags, s)
			}
		}
	}

	var keys []string
	if data.Key != "" {
		parts := strings.Split(data.Key, ",")
		for _, p := range parts {
			if k := strings.TrimSpace(p); k != "" {
				keys = append(keys, k)
			}
		}
	}

	// Helper to split comma-separated values
	splitComma := func(s string) []string {
		if s == "" {
			return nil
		}
		var result []string
		for _, item := range strings.Split(s, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}

	server := domain.Server{
		Alias:                data.Alias,
		Host:                 data.Host,
		User:                 data.User,
		Port:                 port,
		IdentityFiles:        keys,
		Tags:                 tags,
		ProxyJump:            data.ProxyJump,
		ProxyCommand:         data.ProxyCommand,
		RemoteCommand:        data.RemoteCommand,
		RequestTTY:           data.RequestTTY,
		SessionType:          data.SessionType,
		ConnectTimeout:       data.ConnectTimeout,
		ConnectionAttempts:   data.ConnectionAttempts,
		BindAddress:          data.BindAddress,
		BindInterface:        data.BindInterface,
		AddressFamily:        data.AddressFamily,
		ExitOnForwardFailure: data.ExitOnForwardFailure,
		LocalForward:         splitComma(data.LocalForward),
		RemoteForward:        splitComma(data.RemoteForward),
		DynamicForward:       splitComma(data.DynamicForward),
		ClearAllForwardings:  data.ClearAllForwardings,
		// Public key
		PubkeyAuthentication: data.PubkeyAuthentication,
		IdentitiesOnly:       data.IdentitiesOnly,
		// SSH Agent
		AddKeysToAgent: data.AddKeysToAgent,
		IdentityAgent:  data.IdentityAgent,
		// Password & Interactive
		PasswordAuthentication:       data.PasswordAuthentication,
		KbdInteractiveAuthentication: data.KbdInteractiveAuthentication,
		NumberOfPasswordPrompts:      data.NumberOfPasswordPrompts,
		// Advanced
		PreferredAuthentications:    data.PreferredAuthentications,
		ForwardAgent:                data.ForwardAgent,
		ForwardX11:                  data.ForwardX11,
		ForwardX11Trusted:           data.ForwardX11Trusted,
		ControlMaster:               data.ControlMaster,
		ControlPath:                 data.ControlPath,
		ControlPersist:              data.ControlPersist,
		ServerAliveInterval:         data.ServerAliveInterval,
		ServerAliveCountMax:         data.ServerAliveCountMax,
		Compression:                 data.Compression,
		TCPKeepAlive:                data.TCPKeepAlive,
		BatchMode:                   data.BatchMode,
		StrictHostKeyChecking:       data.StrictHostKeyChecking,
		UserKnownHostsFile:          data.UserKnownHostsFile,
		HostKeyAlgorithms:           data.HostKeyAlgorithms,
		PubkeyAcceptedAlgorithms:    data.PubkeyAcceptedAlgorithms,
		HostbasedAcceptedAlgorithms: data.HostbasedAcceptedAlgorithms,
		MACs:                        data.MACs,
		Ciphers:                     data.Ciphers,
		KexAlgorithms:               data.KexAlgorithms,
		VerifyHostKeyDNS:            data.VerifyHostKeyDNS,
		UpdateHostKeys:              data.UpdateHostKeys,
		HashKnownHosts:              data.HashKnownHosts,
		VisualHostKey:               data.VisualHostKey,
		LocalCommand:                data.LocalCommand,
		PermitLocalCommand:          data.PermitLocalCommand,
		EscapeChar:                  data.EscapeChar,
		SendEnv:                     splitComma(data.SendEnv),
		SetEnv:                      splitComma(data.SetEnv),
		LogLevel:                    data.LogLevel,
	}

	// Preserve metadata fields from original if in edit mode
	if sf.mode == ServerFormEdit && sf.original != nil {
		server.PinnedAt = sf.original.PinnedAt
		server.LastSeen = sf.original.LastSeen
		server.SSHCount = sf.original.SSHCount
		// Also preserve Aliases (computed field)
		server.Aliases = sf.original.Aliases
	}

	return server
}

func (sf *ServerForm) OnSave(fn func(domain.Server, *domain.Server)) *ServerForm {
	sf.onSave = fn
	return sf
}

func (sf *ServerForm) OnCancel(fn func()) *ServerForm {
	sf.onCancel = fn
	return sf
}

func (sf *ServerForm) SetApp(app *tview.Application) *ServerForm {
	sf.app = app
	return sf
}

func (sf *ServerForm) SetExistingAliases(aliases []string) *ServerForm {
	sf.existingAliases = aliases
	return sf
}

// SetInitialData sets initial data for pre-filling the form in Add mode
func (sf *ServerForm) SetInitialData(server *domain.Server) *ServerForm {
	sf.initialData = server
	return sf
}

func (sf *ServerForm) SetVersionInfo(version, commit string) *ServerForm {
	sf.version = version
	sf.commit = commit
	// Build the form now that we have version info
	if sf.header == nil {
		sf.build()
	} else {
		// Rebuild header if already exists
		sf.header = NewAppHeader(sf.version, sf.commit, RepoURL)
	}
	return sf
}
