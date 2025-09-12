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
	"regexp"
	"strconv"
	"strings"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ServerFormMode int

const (
	ServerFormAdd ServerFormMode = iota
	ServerFormEdit
)

type ServerForm struct {
	*tview.Flex
	pages      *tview.Pages
	tabBar     *tview.TextView
	forms      map[string]*tview.Form
	currentTab string
	tabs       []string
	mode       ServerFormMode
	original   *domain.Server
	onSave     func(domain.Server, *domain.Server)
	onCancel   func()
}

func NewServerForm(mode ServerFormMode, original *domain.Server) *ServerForm {
	form := &ServerForm{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		pages:    tview.NewPages(),
		tabBar:   tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignCenter).SetRegions(true),
		forms:    make(map[string]*tview.Form),
		mode:     mode,
		original: original,
		tabs: []string{
			"Basic",
			"Connection",
			"Forwarding",
			"Authentication",
			"Multiplexing",
			"Advanced",
		},
	}
	form.currentTab = "Basic"
	form.build()
	return form
}

func (sf *ServerForm) build() {
	title := sf.titleForMode()

	// Create forms for each tab
	sf.createBasicForm()
	sf.createConnectionForm()
	sf.createForwardingForm()
	sf.createAuthenticationForm()
	sf.createMultiplexingForm()
	sf.createAdvancedForm()

	// Setup tab bar
	sf.updateTabBar()

	// Setup layout
	sf.Flex.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	sf.Flex.AddItem(sf.tabBar, 1, 0, false).
		AddItem(sf.pages, 0, 1, true)

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
		return "Edit Server - [yellow]Ctrl+L[white]: Next | [yellow]Ctrl+H[white]: Prev | [yellow]Ctrl+S[white]: Save | [yellow]Esc[white]: Cancel"
	}
	return "Add Server - [yellow]Ctrl+L[white]: Next | [yellow]Ctrl+H[white]: Prev | [yellow]Ctrl+S[white]: Save | [yellow]Esc[white]: Cancel"
}

func (sf *ServerForm) updateTabBar() {
	// Get current tab index
	currentIdx := 0
	for i, tab := range sf.tabs {
		if tab == sf.currentTab {
			currentIdx = i
			break
		}
	}

	// Build tab text with scroll indicator if needed
	var tabText string

	// Check if we need to show scroll indicators
	x, y, width, height := sf.tabBar.GetInnerRect()
	_ = x
	_ = y
	_ = height

	// Calculate the actual visible width needed for all tabs
	// We need to calculate the actual rendered text length without ANSI codes
	estimatedWidth := 0
	for i, tab := range sf.tabs {
		// Each tab has: space + name + space
		estimatedWidth += len(tab) + 2
		if i < len(sf.tabs)-1 {
			// Separator: space + | + space
			estimatedWidth += 3
		}
	}

	// Only show scroll indicators if:
	// 1. Width is known and positive (not 0 from initial render)
	// 2. The estimated width actually exceeds available width
	// When width is 0 or unknown, default to showing all tabs
	showScrollIndicators := width > 20 && estimatedWidth > (width-10)

	if showScrollIndicators {
		// Calculate how many tabs we can fit
		// Reserve space for scroll indicators (4 chars each: "◀ " and " ▶")
		availableWidth := width - 8

		// Calculate how many tabs can fit
		visibleCount := 0
		currentWidth := 0

		for i := 0; i < len(sf.tabs) && currentWidth < availableWidth; i++ {
			tabWidth := len(sf.tabs[i]) + 2
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

		// Ensure at least 2 tabs are visible
		if visibleCount < 2 {
			visibleCount = 2
		}

		// Show scroll indicator on the left if not at first tab
		if currentIdx > 0 {
			tabText = "[gray]◀ [-]"
		}

		// Calculate visible range centered on current tab
		halfVisible := visibleCount / 2
		start := currentIdx - halfVisible + 1
		end := start + visibleCount

		// Adjust boundaries
		if start < 0 {
			start = 0
			end = visibleCount
		}
		if end > len(sf.tabs) {
			end = len(sf.tabs)
			start = end - visibleCount
			if start < 0 {
				start = 0
			}
		}

		// Build visible tabs
		for i := start; i < end && i < len(sf.tabs); i++ {
			tab := sf.tabs[i]
			regionID := fmt.Sprintf("tab_%d", i)
			if tab == sf.currentTab {
				tabText += fmt.Sprintf("[%q][black:white:b] %s [-:-:-][%q] ", regionID, tab, "")
			} else {
				tabText += fmt.Sprintf("[%q][gray::u] %s [-:-:-][%q] ", regionID, tab, "")
			}
			if i < end-1 && i < len(sf.tabs)-1 {
				tabText += "[gray]|[-] "
			}
		}

		// Show scroll indicator on the right if not at last tab
		if currentIdx < len(sf.tabs)-1 {
			tabText += " [gray]▶[-]"
		}
	} else {
		// Show all tabs if they fit
		for i, tab := range sf.tabs {
			regionID := fmt.Sprintf("tab_%d", i)
			if tab == sf.currentTab {
				tabText += fmt.Sprintf("[%q][black:white:b] %s [-:-:-][%q] ", regionID, tab, "")
			} else {
				tabText += fmt.Sprintf("[%q][gray::u] %s [-:-:-][%q] ", regionID, tab, "")
			}
			if i < len(sf.tabs)-1 {
				tabText += "[gray]|[-] "
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

// findOptionIndex finds the index of a value in options slice
func (sf *ServerForm) findOptionIndex(options []string, value string) int {
	for i, opt := range options {
		if strings.EqualFold(opt, value) {
			return i
		}
	}
	return 0 // Default to first option (empty/"")
}

// getDefaultValues returns default form values based on mode
func (sf *ServerForm) getDefaultValues() ServerFormData {
	if sf.mode == ServerFormEdit && sf.original != nil {
		return ServerFormData{
			Alias:                    sf.original.Alias,
			Host:                     sf.original.Host,
			User:                     sf.original.User,
			Port:                     fmt.Sprint(sf.original.Port),
			Key:                      strings.Join(sf.original.IdentityFiles, ", "),
			Tags:                     strings.Join(sf.original.Tags, ", "),
			ProxyJump:                sf.original.ProxyJump,
			ProxyCommand:             sf.original.ProxyCommand,
			RemoteCommand:            sf.original.RemoteCommand,
			RequestTTY:               sf.original.RequestTTY,
			ConnectTimeout:           sf.original.ConnectTimeout,
			ConnectionAttempts:       sf.original.ConnectionAttempts,
			LocalForward:             strings.Join(sf.original.LocalForward, ", "),
			RemoteForward:            strings.Join(sf.original.RemoteForward, ", "),
			DynamicForward:           strings.Join(sf.original.DynamicForward, ", "),
			PubkeyAuthentication:     sf.original.PubkeyAuthentication,
			PasswordAuthentication:   sf.original.PasswordAuthentication,
			PreferredAuthentications: sf.original.PreferredAuthentications,
			IdentitiesOnly:           sf.original.IdentitiesOnly,
			AddKeysToAgent:           sf.original.AddKeysToAgent,
			IdentityAgent:            sf.original.IdentityAgent,
			ForwardAgent:             sf.original.ForwardAgent,
			ForwardX11:               sf.original.ForwardX11,
			ForwardX11Trusted:        sf.original.ForwardX11Trusted,
			ControlMaster:            sf.original.ControlMaster,
			ControlPath:              sf.original.ControlPath,
			ControlPersist:           sf.original.ControlPersist,
			ServerAliveInterval:      sf.original.ServerAliveInterval,
			ServerAliveCountMax:      sf.original.ServerAliveCountMax,
			Compression:              sf.original.Compression,
			TCPKeepAlive:             sf.original.TCPKeepAlive,
			StrictHostKeyChecking:    sf.original.StrictHostKeyChecking,
			UserKnownHostsFile:       sf.original.UserKnownHostsFile,
			HostKeyAlgorithms:        sf.original.HostKeyAlgorithms,
			LocalCommand:             sf.original.LocalCommand,
			PermitLocalCommand:       sf.original.PermitLocalCommand,
			SendEnv:                  strings.Join(sf.original.SendEnv, ", "),
			SetEnv:                   strings.Join(sf.original.SetEnv, ", "),
			LogLevel:                 sf.original.LogLevel,
			BatchMode:                sf.original.BatchMode,
		}
	}
	return ServerFormData{
		User: "root",
		Port: "22",
		Key:  "~/.ssh/id_ed25519",
	}
}

// createBasicForm creates the Basic configuration tab
func (sf *ServerForm) createBasicForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	form.AddInputField("Alias:", defaultValues.Alias, 20, nil, nil)
	form.AddInputField("Host/IP:", defaultValues.Host, 20, nil, nil)
	form.AddInputField("User:", defaultValues.User, 20, nil, nil)
	form.AddInputField("Port:", defaultValues.Port, 20, nil, nil)
	form.AddInputField("Key (comma):", defaultValues.Key, 40, nil, nil)
	form.AddInputField("Tags (comma):", defaultValues.Tags, 30, nil, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
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

	form.AddInputField("ProxyJump:", defaultValues.ProxyJump, 40, nil, nil)
	form.AddInputField("ProxyCommand:", defaultValues.ProxyCommand, 40, nil, nil)
	form.AddInputField("RemoteCommand:", defaultValues.RemoteCommand, 40, nil, nil)

	// RequestTTY dropdown
	requestTTYOptions := []string{"", "yes", "no", "force", "auto"}
	requestTTYIndex := sf.findOptionIndex(requestTTYOptions, defaultValues.RequestTTY)
	form.AddDropDown("RequestTTY:", requestTTYOptions, requestTTYIndex, nil)

	form.AddInputField("ConnectTimeout (seconds):", defaultValues.ConnectTimeout, 10, nil, nil)
	form.AddInputField("ConnectionAttempts:", defaultValues.ConnectionAttempts, 10, nil, nil)
	form.AddInputField("ServerAliveInterval (seconds):", defaultValues.ServerAliveInterval, 10, nil, nil)
	form.AddInputField("ServerAliveCountMax:", defaultValues.ServerAliveCountMax, 20, nil, nil)

	// Compression dropdown
	yesNoOptions := []string{"", "yes", "no"}
	compressionIndex := sf.findOptionIndex(yesNoOptions, defaultValues.Compression)
	form.AddDropDown("Compression:", yesNoOptions, compressionIndex, nil)

	// TCPKeepAlive dropdown
	tcpKeepAliveIndex := sf.findOptionIndex(yesNoOptions, defaultValues.TCPKeepAlive)
	form.AddDropDown("TCPKeepAlive:", yesNoOptions, tcpKeepAliveIndex, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
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
	yesNoOptions := []string{"", "yes", "no"}

	form.AddTextView("[yellow]Port Forwarding[-]", "", 0, 1, true, false)
	form.AddInputField("LocalForward (comma):", defaultValues.LocalForward, 40, nil, nil)
	form.AddInputField("RemoteForward (comma):", defaultValues.RemoteForward, 40, nil, nil)
	form.AddInputField("DynamicForward (comma):", defaultValues.DynamicForward, 40, nil, nil)

	form.AddTextView("[yellow]Agent & X11 Forwarding[-]", "", 0, 1, true, false)

	// ForwardAgent dropdown
	forwardAgentIndex := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardAgent)
	form.AddDropDown("ForwardAgent:", yesNoOptions, forwardAgentIndex, nil)

	// ForwardX11 dropdown
	forwardX11Index := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardX11)
	form.AddDropDown("ForwardX11:", yesNoOptions, forwardX11Index, nil)

	// ForwardX11Trusted dropdown
	forwardX11TrustedIndex := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardX11Trusted)
	form.AddDropDown("ForwardX11Trusted:", yesNoOptions, forwardX11TrustedIndex, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Forwarding"] = form
	sf.pages.AddPage("Forwarding", form, true, false)
}

// createAuthenticationForm creates the Authentication tab
func (sf *ServerForm) createAuthenticationForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()
	yesNoOptions := []string{"", "yes", "no"}

	// PubkeyAuthentication dropdown
	pubkeyIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PubkeyAuthentication)
	form.AddDropDown("PubkeyAuthentication:", yesNoOptions, pubkeyIndex, nil)

	// PasswordAuthentication dropdown
	passwordIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PasswordAuthentication)
	form.AddDropDown("PasswordAuthentication:", yesNoOptions, passwordIndex, nil)

	form.AddInputField("PreferredAuthentications:", defaultValues.PreferredAuthentications, 40, nil, nil)

	// IdentitiesOnly dropdown
	identitiesOnlyIndex := sf.findOptionIndex(yesNoOptions, defaultValues.IdentitiesOnly)
	form.AddDropDown("IdentitiesOnly:", yesNoOptions, identitiesOnlyIndex, nil)

	// AddKeysToAgent dropdown
	addKeysOptions := []string{"", "yes", "no", "ask", "confirm"}
	addKeysIndex := sf.findOptionIndex(addKeysOptions, defaultValues.AddKeysToAgent)
	form.AddDropDown("AddKeysToAgent:", addKeysOptions, addKeysIndex, nil)

	form.AddInputField("IdentityAgent:", defaultValues.IdentityAgent, 40, nil, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Authentication"] = form
	sf.pages.AddPage("Authentication", form, true, false)
}

// createMultiplexingForm creates the Connection Multiplexing tab
func (sf *ServerForm) createMultiplexingForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()

	// ControlMaster dropdown
	controlMasterOptions := []string{"", "yes", "no", "auto", "ask", "autoask"}
	controlMasterIndex := sf.findOptionIndex(controlMasterOptions, defaultValues.ControlMaster)
	form.AddDropDown("ControlMaster:", controlMasterOptions, controlMasterIndex, nil)

	form.AddInputField("ControlPath:", defaultValues.ControlPath, 40, nil, nil)
	form.AddInputField("ControlPersist:", defaultValues.ControlPersist, 20, nil, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
	form.AddButton("Cancel", sf.handleCancel)

	// Set up form-level input capture for shortcuts
	sf.setupFormShortcuts(form)

	sf.forms["Multiplexing"] = form
	sf.pages.AddPage("Multiplexing", form, true, false)
}

// createAdvancedForm creates the Advanced settings tab
func (sf *ServerForm) createAdvancedForm() {
	form := tview.NewForm()
	defaultValues := sf.getDefaultValues()
	yesNoOptions := []string{"", "yes", "no"}

	form.AddTextView("[yellow]Security[-]", "", 0, 1, true, false)

	// StrictHostKeyChecking dropdown
	strictHostKeyOptions := []string{"", "yes", "no", "ask", "accept-new"}
	strictHostKeyIndex := sf.findOptionIndex(strictHostKeyOptions, defaultValues.StrictHostKeyChecking)
	form.AddDropDown("StrictHostKeyChecking:", strictHostKeyOptions, strictHostKeyIndex, nil)

	form.AddInputField("UserKnownHostsFile:", defaultValues.UserKnownHostsFile, 40, nil, nil)
	form.AddInputField("HostKeyAlgorithms:", defaultValues.HostKeyAlgorithms, 40, nil, nil)

	form.AddTextView("[yellow]Command Execution[-]", "", 0, 1, true, false)
	form.AddInputField("LocalCommand:", defaultValues.LocalCommand, 40, nil, nil)

	// PermitLocalCommand dropdown
	permitLocalCommandIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PermitLocalCommand)
	form.AddDropDown("PermitLocalCommand:", yesNoOptions, permitLocalCommandIndex, nil)

	form.AddTextView("[yellow]Environment[-]", "", 0, 1, true, false)
	form.AddInputField("SendEnv (comma):", defaultValues.SendEnv, 40, nil, nil)
	form.AddInputField("SetEnv (comma):", defaultValues.SetEnv, 40, nil, nil)

	form.AddTextView("[yellow]Debugging[-]", "", 0, 1, true, false)

	// LogLevel dropdown
	logLevelOptions := []string{"", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"}
	logLevelIndex := sf.findOptionIndex(logLevelOptions, strings.ToUpper(defaultValues.LogLevel))
	form.AddDropDown("LogLevel:", logLevelOptions, logLevelIndex, nil)

	// BatchMode dropdown
	batchModeIndex := sf.findOptionIndex(yesNoOptions, defaultValues.BatchMode)
	form.AddDropDown("BatchMode:", yesNoOptions, batchModeIndex, nil)

	// Add save and cancel buttons
	form.AddButton("Save", sf.handleSave)
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
	ProxyJump          string
	ProxyCommand       string
	RemoteCommand      string
	RequestTTY         string
	ConnectTimeout     string
	ConnectionAttempts string

	// Port forwarding
	LocalForward   string
	RemoteForward  string
	DynamicForward string

	// Authentication and key management
	PubkeyAuthentication     string
	PasswordAuthentication   string
	PreferredAuthentications string
	IdentitiesOnly           string
	AddKeysToAgent           string
	IdentityAgent            string

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

	// Security settings
	StrictHostKeyChecking string
	UserKnownHostsFile    string
	HostKeyAlgorithms     string

	// Command execution
	LocalCommand       string
	PermitLocalCommand string

	// Environment settings
	SendEnv string
	SetEnv  string

	// Debugging settings
	LogLevel  string
	BatchMode string
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
						return text
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
		Key:   getFieldText("Key"),
		Tags:  getFieldText("Tags"),
		// Connection and proxy settings
		ProxyJump:          getFieldText("ProxyJump:"),
		ProxyCommand:       getFieldText("ProxyCommand:"),
		RemoteCommand:      getFieldText("RemoteCommand:"),
		RequestTTY:         getDropdownValue("RequestTTY:"),
		ConnectTimeout:     getFieldText("ConnectTimeout (seconds):"),
		ConnectionAttempts: getFieldText("ConnectionAttempts:"),
		// Port forwarding
		LocalForward:   getFieldText("LocalForward"),
		RemoteForward:  getFieldText("RemoteForward"),
		DynamicForward: getFieldText("DynamicForward"),
		// Authentication and key management
		PubkeyAuthentication:     getDropdownValue("PubkeyAuthentication:"),
		PasswordAuthentication:   getDropdownValue("PasswordAuthentication:"),
		PreferredAuthentications: getFieldText("PreferredAuthentications:"),
		IdentitiesOnly:           getDropdownValue("IdentitiesOnly:"),
		AddKeysToAgent:           getDropdownValue("AddKeysToAgent:"),
		IdentityAgent:            getFieldText("IdentityAgent:"),
		// Agent and X11 forwarding
		ForwardAgent:      getDropdownValue("ForwardAgent:"),
		ForwardX11:        getDropdownValue("ForwardX11:"),
		ForwardX11Trusted: getDropdownValue("ForwardX11Trusted:"),
		// Connection multiplexing
		ControlMaster:  getDropdownValue("ControlMaster:"),
		ControlPath:    getFieldText("ControlPath:"),
		ControlPersist: getFieldText("ControlPersist:"),
		// Connection reliability settings
		ServerAliveInterval: getFieldText("ServerAliveInterval (seconds):"),
		ServerAliveCountMax: getFieldText("ServerAliveCountMax:"),
		Compression:         getDropdownValue("Compression:"),
		TCPKeepAlive:        getDropdownValue("TCPKeepAlive:"),
		// Security settings
		StrictHostKeyChecking: getDropdownValue("StrictHostKeyChecking:"),
		UserKnownHostsFile:    getFieldText("UserKnownHostsFile:"),
		HostKeyAlgorithms:     getFieldText("HostKeyAlgorithms:"),
		// Command execution
		LocalCommand:       getFieldText("LocalCommand:"),
		PermitLocalCommand: getDropdownValue("PermitLocalCommand:"),
		// Environment settings
		SendEnv: getFieldText("SendEnv"),
		SetEnv:  getFieldText("SetEnv"),
		// Debugging settings
		LogLevel:  strings.ToLower(getDropdownValue("LogLevel:")),
		BatchMode: getDropdownValue("BatchMode:"),
	}
}

func (sf *ServerForm) handleSave() {
	data := sf.getFormData()

	if errMsg := validateServerForm(data); errMsg != "" {
		// Show error in title bar
		sf.Flex.SetTitle(fmt.Sprintf("%s — [red::b]%s[-]", sf.titleForMode(), errMsg))
		sf.Flex.SetBorderColor(tcell.ColorRed)
		return
	}

	// Reset title and border on success
	sf.Flex.SetTitle(sf.titleForMode())
	sf.Flex.SetBorderColor(tcell.Color238)

	server := sf.dataToServer(data)
	if sf.onSave != nil {
		sf.onSave(server, sf.original)
	}
}

func (sf *ServerForm) handleCancel() {
	if sf.onCancel != nil {
		sf.onCancel()
	}
}

func (sf *ServerForm) dataToServer(data ServerFormData) domain.Server {
	port := 22
	if data.Port != "" {
		if n, err := strconv.Atoi(data.Port); err == nil && n > 0 {
			port = n
		}
	}

	var tags []string
	if data.Tags != "" {
		for _, t := range strings.Split(data.Tags, ",") {
			if s := strings.TrimSpace(t); s != "" {
				tags = append(tags, s)
			}
		}
	}

	keys := make([]string, 0)
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

	return domain.Server{
		Alias:                    data.Alias,
		Host:                     data.Host,
		User:                     data.User,
		Port:                     port,
		IdentityFiles:            keys,
		Tags:                     tags,
		ProxyJump:                data.ProxyJump,
		ProxyCommand:             data.ProxyCommand,
		RemoteCommand:            data.RemoteCommand,
		RequestTTY:               data.RequestTTY,
		ConnectTimeout:           data.ConnectTimeout,
		ConnectionAttempts:       data.ConnectionAttempts,
		LocalForward:             splitComma(data.LocalForward),
		RemoteForward:            splitComma(data.RemoteForward),
		DynamicForward:           splitComma(data.DynamicForward),
		PubkeyAuthentication:     data.PubkeyAuthentication,
		PasswordAuthentication:   data.PasswordAuthentication,
		PreferredAuthentications: data.PreferredAuthentications,
		IdentitiesOnly:           data.IdentitiesOnly,
		AddKeysToAgent:           data.AddKeysToAgent,
		IdentityAgent:            data.IdentityAgent,
		ForwardAgent:             data.ForwardAgent,
		ForwardX11:               data.ForwardX11,
		ForwardX11Trusted:        data.ForwardX11Trusted,
		ControlMaster:            data.ControlMaster,
		ControlPath:              data.ControlPath,
		ControlPersist:           data.ControlPersist,
		ServerAliveInterval:      data.ServerAliveInterval,
		ServerAliveCountMax:      data.ServerAliveCountMax,
		Compression:              data.Compression,
		TCPKeepAlive:             data.TCPKeepAlive,
		StrictHostKeyChecking:    data.StrictHostKeyChecking,
		UserKnownHostsFile:       data.UserKnownHostsFile,
		HostKeyAlgorithms:        data.HostKeyAlgorithms,
		LocalCommand:             data.LocalCommand,
		PermitLocalCommand:       data.PermitLocalCommand,
		SendEnv:                  splitComma(data.SendEnv),
		SetEnv:                   splitComma(data.SetEnv),
		LogLevel:                 data.LogLevel,
		BatchMode:                data.BatchMode,
	}
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
