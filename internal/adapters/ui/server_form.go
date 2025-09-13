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
	"net"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// sshDefaults contains the default values for SSH configuration options
// Based on OpenSSH defaults (version 8.x+)
var sshDefaults = map[string]string{
	// Connection settings
	"Port":                      "22",
	"ConnectTimeout":            "none",
	"ConnectionAttempts":        "1",
	"TCPKeepAlive":              "yes",
	"ServerAliveInterval":       "0",
	"ServerAliveCountMax":       "3",
	"Compression":               "no",
	"AddressFamily":             "any",
	"ExitOnForwardFailure":      "no",
	"IPQoS":                     "af21 cs1",
	"CanonicalizeHostname":      "no",
	"CanonicalizeFallbackLocal": "yes",
	"CanonicalizeMaxDots":       "1",

	// Authentication
	"PubkeyAuthentication":         "yes",
	"PasswordAuthentication":       "yes",
	"IdentitiesOnly":               "no",
	"AddKeysToAgent":               "no",
	"KbdInteractiveAuthentication": "yes",
	"NumberOfPasswordPrompts":      "3",

	// Forwarding
	"ForwardAgent":        "no",
	"ForwardX11":          "no",
	"ForwardX11Trusted":   "no",
	"ClearAllForwardings": "no",
	"GatewayPorts":        "no",

	// Multiplexing
	"ControlMaster":  "no",
	"ControlPath":    "none",
	"ControlPersist": "no",

	// Security
	"StrictHostKeyChecking": "ask",
	"CheckHostIP":           "no",
	"FingerprintHash":       "SHA256",
	"VerifyHostKeyDNS":      "no",
	"UpdateHostKeys":        "no",
	"HashKnownHosts":        "no",
	"VisualHostKey":         "no",
	"PermitLocalCommand":    "no",
	"EscapeChar":            "~",
	"BatchMode":             "no",

	// Other
	"RequestTTY":  "auto",
	"SessionType": "default",
	"LogLevel":    "INFO",
}

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
	*tview.Flex             // The root container (includes header, form panel and hint bar)
	header      *AppHeader  // The app header
	formPanel   *tview.Flex // The actual form panel
	pages       *tview.Pages
	tabBar      *tview.TextView
	forms       map[string]*tview.Form
	currentTab  string
	tabs        []string
	tabAbbrev   map[string]string // Abbreviated tab names for narrow views
	mode        ServerFormMode
	original    *domain.Server
	onSave      func(domain.Server, *domain.Server)
	onCancel    func()
	app         *tview.Application // Reference to app for showing modals
	version     string             // Version for header
	commit      string             // Commit for header
}

func NewServerForm(mode ServerFormMode, original *domain.Server) *ServerForm {
	form := &ServerForm{
		Flex:      tview.NewFlex().SetDirection(tview.FlexRow),
		formPanel: tview.NewFlex().SetDirection(tview.FlexRow),
		pages:     tview.NewPages(),
		tabBar:    tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).SetRegions(true),
		forms:     make(map[string]*tview.Form),
		mode:      mode,
		original:  original,
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
	title := sf.titleForMode()

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
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	sf.formPanel.AddItem(sf.tabBar, 1, 0, false).
		AddItem(sf.pages, 0, 1, true)

	// Create hint bar with same background as main screen's status bar
	hintBar := tview.NewTextView().SetDynamicColors(true)
	hintBar.SetBackgroundColor(tcell.Color235)
	hintBar.SetTextAlign(tview.AlignCenter)
	hintBar.SetText("[white]^H/^L[-] Navigate  • [white]^S[-] Save  • [white]Esc[-] Cancel")

	// Setup main container - header at top, hint bar at bottom
	sf.Flex.AddItem(sf.header, 2, 0, false).
		AddItem(sf.formPanel, 0, 1, true).
		AddItem(hintBar, 1, 0, false)

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
		if tab == tabName {
			sf.currentTab = tabName
			sf.pages.SwitchToPage(tabName)
			sf.updateTabBar()
			break
		}
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

func (sf *ServerForm) setupKeyboardShortcuts() {
	// Set input capture for the main flex container
	sf.Flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
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

// getDefaultValues returns default form values based on mode
func (sf *ServerForm) getDefaultValues() ServerFormData {
	if sf.mode == ServerFormEdit && sf.original != nil {
		return ServerFormData{
			Alias:                sf.original.Alias,
			Host:                 sf.original.Host,
			User:                 sf.original.User,
			Port:                 fmt.Sprint(sf.original.Port),
			Key:                  strings.Join(sf.original.IdentityFiles, ", "),
			Tags:                 strings.Join(sf.original.Tags, ", "),
			ProxyJump:            sf.original.ProxyJump,
			ProxyCommand:         sf.original.ProxyCommand,
			RemoteCommand:        sf.original.RemoteCommand,
			RequestTTY:           sf.original.RequestTTY,
			SessionType:          sf.original.SessionType,
			ConnectTimeout:       sf.original.ConnectTimeout,
			ConnectionAttempts:   sf.original.ConnectionAttempts,
			BindAddress:          sf.original.BindAddress,
			BindInterface:        sf.original.BindInterface,
			AddressFamily:        sf.original.AddressFamily,
			ExitOnForwardFailure: sf.original.ExitOnForwardFailure,
			IPQoS:                sf.original.IPQoS,
			// Hostname canonicalization
			CanonicalizeHostname:        sf.original.CanonicalizeHostname,
			CanonicalDomains:            sf.original.CanonicalDomains,
			CanonicalizeFallbackLocal:   sf.original.CanonicalizeFallbackLocal,
			CanonicalizeMaxDots:         sf.original.CanonicalizeMaxDots,
			CanonicalizePermittedCNAMEs: sf.original.CanonicalizePermittedCNAMEs,
			GatewayPorts:                sf.original.GatewayPorts,
			LocalForward:                strings.Join(sf.original.LocalForward, ", "),
			RemoteForward:               strings.Join(sf.original.RemoteForward, ", "),
			DynamicForward:              strings.Join(sf.original.DynamicForward, ", "),
			ClearAllForwardings:         sf.original.ClearAllForwardings,
			// Public key
			PubkeyAuthentication: sf.original.PubkeyAuthentication,
			IdentitiesOnly:       sf.original.IdentitiesOnly,
			// SSH Agent
			AddKeysToAgent: sf.original.AddKeysToAgent,
			IdentityAgent:  sf.original.IdentityAgent,
			// Password & Interactive
			PasswordAuthentication:       sf.original.PasswordAuthentication,
			KbdInteractiveAuthentication: sf.original.KbdInteractiveAuthentication,
			NumberOfPasswordPrompts:      sf.original.NumberOfPasswordPrompts,
			// Advanced
			PreferredAuthentications:    sf.original.PreferredAuthentications,
			ForwardAgent:                sf.original.ForwardAgent,
			ForwardX11:                  sf.original.ForwardX11,
			ForwardX11Trusted:           sf.original.ForwardX11Trusted,
			ControlMaster:               sf.original.ControlMaster,
			ControlPath:                 sf.original.ControlPath,
			ControlPersist:              sf.original.ControlPersist,
			ServerAliveInterval:         sf.original.ServerAliveInterval,
			ServerAliveCountMax:         sf.original.ServerAliveCountMax,
			Compression:                 sf.original.Compression,
			TCPKeepAlive:                sf.original.TCPKeepAlive,
			BatchMode:                   sf.original.BatchMode,
			StrictHostKeyChecking:       sf.original.StrictHostKeyChecking,
			UserKnownHostsFile:          sf.original.UserKnownHostsFile,
			HostKeyAlgorithms:           sf.original.HostKeyAlgorithms,
			PubkeyAcceptedAlgorithms:    sf.original.PubkeyAcceptedAlgorithms,
			HostbasedAcceptedAlgorithms: sf.original.HostbasedAcceptedAlgorithms,
			MACs:                        sf.original.MACs,
			Ciphers:                     sf.original.Ciphers,
			KexAlgorithms:               sf.original.KexAlgorithms,
			VerifyHostKeyDNS:            sf.original.VerifyHostKeyDNS,
			UpdateHostKeys:              sf.original.UpdateHostKeys,
			HashKnownHosts:              sf.original.HashKnownHosts,
			VisualHostKey:               sf.original.VisualHostKey,
			LocalCommand:                sf.original.LocalCommand,
			PermitLocalCommand:          sf.original.PermitLocalCommand,
			EscapeChar:                  sf.original.EscapeChar,
			SendEnv:                     strings.Join(sf.original.SendEnv, ", "),
			SetEnv:                      strings.Join(sf.original.SetEnv, ", "),
			LogLevel:                    sf.original.LogLevel,
		}
	}
	// Determine default key based on availability
	defaultKey := "~/.ssh/id_ed25519"
	availableKeys := GetAvailableSSHKeys()
	if len(availableKeys) > 0 {
		// Use the first available key as default
		defaultKey = availableKeys[0]
	} else {
		// Check if id_ed25519 exists, otherwise try id_rsa
		for _, key := range []string{"~/.ssh/id_ed25519", "~/.ssh/id_rsa"} {
			if homeDir, err := os.UserHomeDir(); err == nil {
				keyPath := strings.Replace(key, "~", homeDir, 1)
				if _, err := os.Stat(keyPath); err == nil {
					defaultKey = key
					break
				}
			}
		}
	}

	return ServerFormData{
		User: "root",
		Port: "22",
		Key:  defaultKey,
	}
}

// createBasicForm creates the Basic configuration tab
func (sf *ServerForm) createBasicForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	form.AddInputField("Alias:", defaultValues.Alias, 20, nil, nil)
	form.AddInputField("Host/IP:", defaultValues.Host, 20, nil, nil)

	userField := tview.NewInputField().
		SetLabel("User:").
		SetText(defaultValues.User).
		SetFieldWidth(20).
		SetPlaceholder("default: root")
	form.AddFormItem(userField)

	portField := tview.NewInputField().
		SetLabel("Port:").
		SetText(defaultValues.Port).
		SetFieldWidth(20).
		SetPlaceholder("default: 22")
	form.AddFormItem(portField)
	keysField := tview.NewInputField().
		SetLabel("Keys:").
		SetText(defaultValues.Key).
		SetFieldWidth(40).
		SetPlaceholder("e.g., ~/.ssh/id_rsa, ~/.ssh/id_ed25519")
	keysField.SetAutocompleteFunc(sf.createSSHKeyAutocomplete())
	form.AddFormItem(keysField)

	tagsField := tview.NewInputField().
		SetLabel("Tags:").
		SetText(defaultValues.Tags).
		SetFieldWidth(30).
		SetPlaceholder("comma-separated tags")
	form.AddFormItem(tagsField)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveWrapper)
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
	form.AddInputField("ProxyJump:", defaultValues.ProxyJump, 40, nil, nil)
	form.AddInputField("ProxyCommand:", defaultValues.ProxyCommand, 40, nil, nil)
	remoteCommandField := tview.NewInputField().
		SetLabel("RemoteCommand:").
		SetText(defaultValues.RemoteCommand).
		SetFieldWidth(40).
		SetPlaceholder("command to run, or 'none' to clear (OpenSSH 7.6+)")
	form.AddFormItem(remoteCommandField)

	// RequestTTY dropdown
	requestTTYOptions := createOptionsWithDefault("RequestTTY", []string{"", "yes", "no", "force", "auto"})
	requestTTYIndex := sf.findOptionIndex(requestTTYOptions, defaultValues.RequestTTY)
	form.AddDropDown("RequestTTY:", requestTTYOptions, requestTTYIndex, nil)

	// SessionType dropdown (OpenSSH 8.7+)
	sessionTypeOptions := createOptionsWithDefault("SessionType", []string{"", "none (-N)", "subsystem (-s)", "default"})
	sessionTypeIndex := sf.findOptionIndex(sessionTypeOptions, defaultValues.SessionType)
	form.AddDropDown("SessionType:", sessionTypeOptions, sessionTypeIndex, nil)

	form.AddTextView("\n[yellow]▶ Connection Settings[-]", "", 0, 1, true, false)
	connectTimeoutField := tview.NewInputField().
		SetLabel("ConnectTimeout:").
		SetText(defaultValues.ConnectTimeout).
		SetFieldWidth(10).
		SetPlaceholder("seconds (default: none)")
	form.AddFormItem(connectTimeoutField)

	connectionAttemptsField := tview.NewInputField().
		SetLabel("ConnectionAttempts:").
		SetText(defaultValues.ConnectionAttempts).
		SetFieldWidth(10).
		SetPlaceholder("default: 1")
	form.AddFormItem(connectionAttemptsField)

	// IPQoS field (moved from Bind Options)
	ipqosField := tview.NewInputField().
		SetLabel("IPQoS:").
		SetText(defaultValues.IPQoS).
		SetFieldWidth(20).
		SetPlaceholder("default: af21 cs1")
	form.AddFormItem(ipqosField)

	// BatchMode dropdown (moved from Keep-Alive)
	batchModeOptions := createOptionsWithDefault("BatchMode", []string{"", "yes", "no"})
	batchModeIndex := sf.findOptionIndex(batchModeOptions, defaultValues.BatchMode)
	form.AddDropDown("BatchMode:", batchModeOptions, batchModeIndex, nil)

	form.AddTextView("\n[yellow]▶ Bind Options[-]", "", 0, 1, true, false)
	form.AddInputField("BindAddress:", defaultValues.BindAddress, 40, nil, nil)

	// BindInterface dropdown with available network interfaces
	interfaceOptions := append([]string{""}, GetNetworkInterfaces()...)
	bindInterfaceIndex := sf.findOptionIndex(interfaceOptions, defaultValues.BindInterface)
	form.AddDropDown("BindInterface:", interfaceOptions, bindInterfaceIndex, nil)

	// AddressFamily dropdown
	addressFamilyOptions := createOptionsWithDefault("AddressFamily", []string{"", "any", "inet", "inet6"})
	addressFamilyIndex := sf.findOptionIndex(addressFamilyOptions, defaultValues.AddressFamily)
	form.AddDropDown("AddressFamily:", addressFamilyOptions, addressFamilyIndex, nil)

	form.AddTextView("\n[yellow]▶ Hostname Canonicalization[-]", "", 0, 1, true, false)

	// CanonicalizeHostname dropdown
	canonicalizeOptions := createOptionsWithDefault("CanonicalizeHostname", []string{"", "yes", "no", "always"})
	canonicalizeIndex := sf.findOptionIndex(canonicalizeOptions, defaultValues.CanonicalizeHostname)
	form.AddDropDown("CanonicalizeHostname:", canonicalizeOptions, canonicalizeIndex, nil)

	canonicalDomainsField := tview.NewInputField().
		SetLabel("CanonicalDomains:").
		SetText(defaultValues.CanonicalDomains).
		SetFieldWidth(40).
		SetPlaceholder("e.g., example.com, internal.net")
	form.AddFormItem(canonicalDomainsField)

	// CanonicalizeFallbackLocal dropdown
	fallbackOptions := createOptionsWithDefault("CanonicalizeFallbackLocal", []string{"", "yes", "no"})
	fallbackIndex := sf.findOptionIndex(fallbackOptions, defaultValues.CanonicalizeFallbackLocal)
	form.AddDropDown("CanonicalizeFallbackLocal:", fallbackOptions, fallbackIndex, nil)

	canonicalizeMaxDotsField := tview.NewInputField().
		SetLabel("CanonicalizeMaxDots:").
		SetText(defaultValues.CanonicalizeMaxDots).
		SetFieldWidth(10).
		SetPlaceholder("default: 1")
	form.AddFormItem(canonicalizeMaxDotsField)

	canonicalizePermittedCNAMEsField := tview.NewInputField().
		SetLabel("CanonicalizePermittedCNAMEs:").
		SetText(defaultValues.CanonicalizePermittedCNAMEs).
		SetFieldWidth(40).
		SetPlaceholder("e.g., *.example.com:example.net")
	form.AddFormItem(canonicalizePermittedCNAMEsField)

	form.AddTextView("\n[yellow]▶ Keep-Alive[-]", "", 0, 1, true, false)
	serverAliveIntervalField := tview.NewInputField().
		SetLabel("ServerAliveInterval:").
		SetText(defaultValues.ServerAliveInterval).
		SetFieldWidth(10).
		SetPlaceholder("seconds (default: 0)")
	form.AddFormItem(serverAliveIntervalField)

	serverAliveCountMaxField := tview.NewInputField().
		SetLabel("ServerAliveCountMax:").
		SetText(defaultValues.ServerAliveCountMax).
		SetFieldWidth(10).
		SetPlaceholder("default: 3")
	form.AddFormItem(serverAliveCountMaxField)

	// Compression dropdown
	compressionOptions := createOptionsWithDefault("Compression", []string{"", "yes", "no"})
	compressionIndex := sf.findOptionIndex(compressionOptions, defaultValues.Compression)
	form.AddDropDown("Compression:", compressionOptions, compressionIndex, nil)

	// TCPKeepAlive dropdown
	tcpKeepAliveOptions := createOptionsWithDefault("TCPKeepAlive", []string{"", "yes", "no"})
	tcpKeepAliveIndex := sf.findOptionIndex(tcpKeepAliveOptions, defaultValues.TCPKeepAlive)
	form.AddDropDown("TCPKeepAlive:", tcpKeepAliveOptions, tcpKeepAliveIndex, nil)

	form.AddTextView("\n[yellow]▶ Multiplexing[-]", "", 0, 1, true, false)
	// ControlMaster dropdown
	controlMasterOptions := createOptionsWithDefault("ControlMaster", []string{"", "yes", "no", "auto", "ask", "autoask"})
	controlMasterIndex := sf.findOptionIndex(controlMasterOptions, defaultValues.ControlMaster)
	form.AddDropDown("ControlMaster:", controlMasterOptions, controlMasterIndex, nil)
	form.AddInputField("ControlPath:", defaultValues.ControlPath, 40, nil, nil)
	form.AddInputField("ControlPersist:", defaultValues.ControlPersist, 20, nil, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveWrapper)
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
	localForwardField := tview.NewInputField().
		SetLabel("LocalForward:").
		SetText(defaultValues.LocalForward).
		SetFieldWidth(40).
		SetPlaceholder("e.g., 8080:localhost:80, 3000:localhost:3000")
	form.AddFormItem(localForwardField)

	remoteForwardField := tview.NewInputField().
		SetLabel("RemoteForward:").
		SetText(defaultValues.RemoteForward).
		SetFieldWidth(40).
		SetPlaceholder("e.g., 80:localhost:8080")
	form.AddFormItem(remoteForwardField)

	dynamicForwardField := tview.NewInputField().
		SetLabel("DynamicForward:").
		SetText(defaultValues.DynamicForward).
		SetFieldWidth(40).
		SetPlaceholder("e.g., 1080, 1081")
	form.AddFormItem(dynamicForwardField)

	// ClearAllForwardings dropdown
	clearAllForwardingsOptions := createOptionsWithDefault("ClearAllForwardings", []string{"", "yes", "no"})
	clearAllForwardingsIndex := sf.findOptionIndex(clearAllForwardingsOptions, defaultValues.ClearAllForwardings)
	form.AddDropDown("ClearAllForwardings:", clearAllForwardingsOptions, clearAllForwardingsIndex, nil)

	// ExitOnForwardFailure dropdown
	exitOnForwardFailureOptions := createOptionsWithDefault("ExitOnForwardFailure", []string{"", "yes", "no"})
	exitOnForwardFailureIndex := sf.findOptionIndex(exitOnForwardFailureOptions, defaultValues.ExitOnForwardFailure)
	form.AddDropDown("ExitOnForwardFailure:", exitOnForwardFailureOptions, exitOnForwardFailureIndex, nil)

	// GatewayPorts dropdown
	gatewayPortsOptions := createOptionsWithDefault("GatewayPorts", []string{"", "yes", "no", "clientspecified"})
	gatewayPortsIndex := sf.findOptionIndex(gatewayPortsOptions, defaultValues.GatewayPorts)
	form.AddDropDown("GatewayPorts:", gatewayPortsOptions, gatewayPortsIndex, nil)

	form.AddTextView("\n[yellow]▶ Agent & X11 Forwarding[-]", "", 0, 1, true, false)

	// ForwardAgent dropdown
	forwardAgentOptions := createOptionsWithDefault("ForwardAgent", []string{"", "yes", "no"})
	forwardAgentIndex := sf.findOptionIndex(forwardAgentOptions, defaultValues.ForwardAgent)
	form.AddDropDown("ForwardAgent:", forwardAgentOptions, forwardAgentIndex, nil)

	// ForwardX11 dropdown
	forwardX11Options := createOptionsWithDefault("ForwardX11", []string{"", "yes", "no"})
	forwardX11Index := sf.findOptionIndex(forwardX11Options, defaultValues.ForwardX11)
	form.AddDropDown("ForwardX11:", forwardX11Options, forwardX11Index, nil)

	// ForwardX11Trusted dropdown
	forwardX11TrustedOptions := createOptionsWithDefault("ForwardX11Trusted", []string{"", "yes", "no"})
	forwardX11TrustedIndex := sf.findOptionIndex(forwardX11TrustedOptions, defaultValues.ForwardX11Trusted)
	form.AddDropDown("ForwardX11Trusted:", forwardX11TrustedOptions, forwardX11TrustedIndex, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveWrapper)
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
	form.AddDropDown("PubkeyAuthentication:", pubkeyOptions, pubkeyIndex, nil)

	// IdentitiesOnly dropdown - controls whether to use only specified identity files
	identitiesOnlyOptions := createOptionsWithDefault("IdentitiesOnly", []string{"", "yes", "no"})
	identitiesOnlyIndex := sf.findOptionIndex(identitiesOnlyOptions, defaultValues.IdentitiesOnly)
	form.AddDropDown("IdentitiesOnly:", identitiesOnlyOptions, identitiesOnlyIndex, nil)

	// SSH Agent settings
	form.AddTextView("\n[yellow]▶ SSH Agent[-]", "", 0, 1, true, false)

	// AddKeysToAgent dropdown
	addKeysOptions := createOptionsWithDefault("AddKeysToAgent", []string{"", "yes", "no", "ask", "confirm"})
	addKeysIndex := sf.findOptionIndex(addKeysOptions, defaultValues.AddKeysToAgent)
	form.AddDropDown("AddKeysToAgent:", addKeysOptions, addKeysIndex, nil)

	form.AddInputField("IdentityAgent:", defaultValues.IdentityAgent, 40, nil, nil)

	// Password/Interactive authentication
	form.AddTextView("\n[yellow]▶ Password & Interactive[-]", "", 0, 1, true, false)

	// PasswordAuthentication dropdown
	passwordOptions := createOptionsWithDefault("PasswordAuthentication", []string{"", "yes", "no"})
	passwordIndex := sf.findOptionIndex(passwordOptions, defaultValues.PasswordAuthentication)
	form.AddDropDown("PasswordAuthentication:", passwordOptions, passwordIndex, nil)

	// KbdInteractiveAuthentication dropdown
	kbdInteractiveOptions := createOptionsWithDefault("KbdInteractiveAuthentication", []string{"", "yes", "no"})
	kbdInteractiveIndex := sf.findOptionIndex(kbdInteractiveOptions, defaultValues.KbdInteractiveAuthentication)
	form.AddDropDown("KbdInteractiveAuthentication:", kbdInteractiveOptions, kbdInteractiveIndex, nil)

	// NumberOfPasswordPrompts field
	passwordPromptsField := tview.NewInputField().
		SetLabel("NumberOfPasswordPrompts:").
		SetText(defaultValues.NumberOfPasswordPrompts).
		SetFieldWidth(10).
		SetPlaceholder("default: 3")
	form.AddFormItem(passwordPromptsField)

	// Advanced: Authentication order preference
	form.AddTextView("\n[yellow]▶ Advanced[-]", "", 0, 1, true, false)

	preferredAuthField := tview.NewInputField().
		SetLabel("PreferredAuthentications:").
		SetText(defaultValues.PreferredAuthentications).
		SetFieldWidth(40).
		SetPlaceholder("e.g., publickey,password")
	form.AddFormItem(preferredAuthField)

	// PubkeyAcceptedAlgorithms with autocomplete support (moved from Advanced/Cryptography)
	pubkeyAlgField := tview.NewInputField().
		SetLabel("PubkeyAcceptedAlgorithms:").
		SetText(defaultValues.PubkeyAcceptedAlgorithms).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	pubkeyAlgField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(pubkeyAlgorithms))
	form.AddFormItem(pubkeyAlgField)

	// HostbasedAcceptedAlgorithms with autocomplete support (moved from Advanced/Cryptography)
	hostbasedAlgField := tview.NewInputField().
		SetLabel("HostbasedAcceptedAlgorithms:").
		SetText(defaultValues.HostbasedAcceptedAlgorithms).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	hostbasedAlgField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(pubkeyAlgorithms))
	form.AddFormItem(hostbasedAlgField)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveWrapper)
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
	form.AddDropDown("StrictHostKeyChecking:", strictHostKeyOptions, strictHostKeyIndex, nil)

	// CheckHostIP dropdown
	checkHostIPOptions := createOptionsWithDefault("CheckHostIP", []string{"", "yes", "no"})
	checkHostIPIndex := sf.findOptionIndex(checkHostIPOptions, defaultValues.CheckHostIP)
	form.AddDropDown("CheckHostIP:", checkHostIPOptions, checkHostIPIndex, nil)

	// FingerprintHash dropdown
	fingerprintHashOptions := createOptionsWithDefault("FingerprintHash", []string{"", "md5", "sha256"})
	fingerprintHashIndex := sf.findOptionIndex(fingerprintHashOptions, defaultValues.FingerprintHash)
	form.AddDropDown("FingerprintHash:", fingerprintHashOptions, fingerprintHashIndex, nil)

	// VerifyHostKeyDNS dropdown
	verifyHostKeyDNSOptions := createOptionsWithDefault("VerifyHostKeyDNS", []string{"", "yes", "no", "ask"})
	verifyHostKeyDNSIndex := sf.findOptionIndex(verifyHostKeyDNSOptions, defaultValues.VerifyHostKeyDNS)
	form.AddDropDown("VerifyHostKeyDNS:", verifyHostKeyDNSOptions, verifyHostKeyDNSIndex, nil)

	// UpdateHostKeys dropdown
	updateHostKeysOptions := createOptionsWithDefault("UpdateHostKeys", []string{"", "yes", "no", "ask"})
	updateHostKeysIndex := sf.findOptionIndex(updateHostKeysOptions, defaultValues.UpdateHostKeys)
	form.AddDropDown("UpdateHostKeys:", updateHostKeysOptions, updateHostKeysIndex, nil)

	// HashKnownHosts dropdown
	hashKnownHostsOptions := createOptionsWithDefault("HashKnownHosts", []string{"", "yes", "no"})
	hashKnownHostsIndex := sf.findOptionIndex(hashKnownHostsOptions, defaultValues.HashKnownHosts)
	form.AddDropDown("HashKnownHosts:", hashKnownHostsOptions, hashKnownHostsIndex, nil)

	// VisualHostKey dropdown
	visualHostKeyOptions := createOptionsWithDefault("VisualHostKey", []string{"", "yes", "no"})
	visualHostKeyIndex := sf.findOptionIndex(visualHostKeyOptions, defaultValues.VisualHostKey)
	form.AddDropDown("VisualHostKey:", visualHostKeyOptions, visualHostKeyIndex, nil)

	form.AddInputField("UserKnownHostsFile:", defaultValues.UserKnownHostsFile, 40, nil, nil)

	form.AddTextView("\n[yellow]▶ Cryptography[-]", "", 0, 1, true, false)

	// Ciphers with autocomplete support
	ciphersField := tview.NewInputField().
		SetLabel("Ciphers:").
		SetText(defaultValues.Ciphers).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	ciphersField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(cipherAlgorithms))
	form.AddFormItem(ciphersField)

	// MACs with autocomplete support
	macsField := tview.NewInputField().
		SetLabel("MACs:").
		SetText(defaultValues.MACs).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	macsField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(macAlgorithms))
	form.AddFormItem(macsField)

	// KexAlgorithms with autocomplete support
	kexField := tview.NewInputField().
		SetLabel("KexAlgorithms:").
		SetText(defaultValues.KexAlgorithms).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	kexField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(kexAlgorithms))
	form.AddFormItem(kexField)

	// HostKeyAlgorithms with autocomplete support
	hostKeyField := tview.NewInputField().
		SetLabel("HostKeyAlgorithms:").
		SetText(defaultValues.HostKeyAlgorithms).
		SetFieldWidth(40).
		SetPlaceholder("algorithms (+/-/^ prefix supported)")
	hostKeyField.SetAutocompleteFunc(sf.createAlgorithmAutocomplete(hostKeyAlgorithms))
	form.AddFormItem(hostKeyField)

	form.AddTextView("\n[yellow]▶ Command Execution[-]", "", 0, 1, true, false)
	form.AddInputField("LocalCommand:", defaultValues.LocalCommand, 40, nil, nil)

	// PermitLocalCommand dropdown
	permitLocalCommandOptions := createOptionsWithDefault("PermitLocalCommand", []string{"", "yes", "no"})
	permitLocalCommandIndex := sf.findOptionIndex(permitLocalCommandOptions, defaultValues.PermitLocalCommand)
	form.AddDropDown("PermitLocalCommand:", permitLocalCommandOptions, permitLocalCommandIndex, nil)

	// EscapeChar input field
	escapeCharField := tview.NewInputField().
		SetLabel("EscapeChar:").
		SetText(defaultValues.EscapeChar).
		SetFieldWidth(10).
		SetPlaceholder("default: ~")
	form.AddFormItem(escapeCharField)

	form.AddTextView("\n[yellow]▶ Environment[-]", "", 0, 1, true, false)
	sendEnvField := tview.NewInputField().
		SetLabel("SendEnv:").
		SetText(defaultValues.SendEnv).
		SetFieldWidth(40).
		SetPlaceholder("e.g., LANG, LC_*, TERM")
	form.AddFormItem(sendEnvField)

	setEnvField := tview.NewInputField().
		SetLabel("SetEnv:").
		SetText(defaultValues.SetEnv).
		SetFieldWidth(40).
		SetPlaceholder("e.g., FOO=bar, DEBUG=1")
	form.AddFormItem(setEnvField)

	form.AddTextView("\n[yellow]▶ Debugging[-]", "", 0, 1, true, false)

	// LogLevel dropdown
	logLevelOptions := createOptionsWithDefault("LogLevel", []string{"", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"})
	logLevelIndex := sf.findOptionIndex(logLevelOptions, defaultValues.LogLevel)
	form.AddDropDown("LogLevel:", logLevelOptions, logLevelIndex, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSaveWrapper)
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

func (sf *ServerForm) getFormData() ServerFormData {
	// Helper function to get text from InputField across all forms
	getFieldText := func(fieldName string) string {
		for _, form := range sf.forms {
			for i := 0; i < form.GetFormItemCount(); i++ {
				if field, ok := form.GetFormItem(i).(*tview.InputField); ok {
					label := strings.TrimSpace(field.GetLabel())
					if strings.HasPrefix(label, fieldName) {
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
					if strings.HasPrefix(label, fieldName) {
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

func (sf *ServerForm) handleSave() bool {
	data := sf.getFormData()

	if errMsg := validateServerForm(data); errMsg != "" {
		// Show error in form panel title bar
		sf.formPanel.SetTitle(fmt.Sprintf("%s — [red::b]%s[-]", sf.titleForMode(), errMsg))
		sf.formPanel.SetBorderColor(tcell.ColorRed)
		return false // Validation failed
	}

	// Reset title and border on success
	sf.formPanel.SetTitle(sf.titleForMode())
	sf.formPanel.SetBorderColor(tcell.Color238)

	server := sf.dataToServer(data)
	if sf.onSave != nil {
		sf.onSave(server, sf.original)
	}
	return true // Save successful
}

// handleSaveWrapper is used for button callbacks that don't need return value
func (sf *ServerForm) handleSaveWrapper() {
	sf.handleSave()
}

func (sf *ServerForm) handleCancel() {
	// Check if there are unsaved changes
	if sf.hasUnsavedChanges() {
		// If app reference is available, show confirmation dialog
		if sf.app != nil {
			modal := tview.NewModal().
				SetText("You have unsaved changes. Are you sure you want to exit?").
				AddButtons([]string{"(S)ave", "(D)iscard", "(C)ancel"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					switch buttonLabel {
					case "(S)ave":
						// Try to save, if successful it will exit
						if sf.handleSave() {
							// Save successful, modal will be replaced by onSave callback
						} else {
							// Validation failed, return to form
							sf.app.SetRoot(sf.Flex, true)
						}
					case "(D)iscard":
						if sf.onCancel != nil {
							sf.onCancel()
						}
					case "(C)ancel":
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
	// If creating new server, any non-empty required fields mean changes
	if sf.mode == ServerFormAdd {
		data := sf.getFormData()
		return data.Alias != "" || data.Host != "" || data.User != ""
	}

	// If editing, compare with original
	if sf.original == nil {
		return false
	}

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

// validateServerForm returns an error message string if validation fails; empty string means valid.
func validateServerForm(data ServerFormData) string {
	alias := data.Alias
	if alias == "" {
		return "Alias is required"
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_.-]+$`).MatchString(alias) {
		return "Alias may contain letters, digits, dot, dash, underscore"
	}

	host := data.Host
	if host == "" {
		return "Host/IP is required"
	}
	if ip := net.ParseIP(host); ip == nil {

		if strings.Contains(host, " ") {
			return "Host must not contain spaces"
		}
		if !regexp.MustCompile(`^[A-Za-z0-9.-]+$`).MatchString(host) {
			return "Host contains invalid characters"
		}
		if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
			return "Host must not start or end with a dot"
		}
		for _, lbl := range strings.Split(host, ".") {
			if lbl == "" {
				return "Host must not contain empty labels"
			}
			if strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
				return "Hostname labels must not start or end with a hyphen"
			}
		}
	}

	if data.Port != "" {
		p, err := strconv.Atoi(data.Port)
		if err != nil || p < 1 || p > 65535 {
			return "Port must be a number between 1 and 65535"
		}
	}

	return ""
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
