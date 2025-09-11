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
	*tview.Form
	mode     ServerFormMode
	original *domain.Server
	onSave   func(domain.Server, *domain.Server)
	onCancel func()
}

func NewServerForm(mode ServerFormMode, original *domain.Server) *ServerForm {
	form := &ServerForm{
		Form:     tview.NewForm(),
		mode:     mode,
		original: original,
	}
	form.build()
	return form
}

func (sf *ServerForm) build() {
	title := sf.titleForMode()

	sf.Form.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	sf.addFormFields()

	sf.Form.AddButton("Save", sf.handleSave)
	sf.Form.AddButton("Cancel", sf.handleCancel)
	sf.Form.SetCancelFunc(sf.handleCancel)
}

func (sf *ServerForm) titleForMode() string {
	if sf.mode == ServerFormEdit {
		return "Edit Server"
	}
	return "Add Server"
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

func (sf *ServerForm) addFormFields() {
	var defaultValues ServerFormData
	if sf.mode == ServerFormEdit && sf.original != nil {
		defaultValues = ServerFormData{
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
	} else {
		defaultValues = ServerFormData{
			User: "root",
			Port: "22",
			Key:  "~/.ssh/id_ed25519",
		}
	}

	// Basic configuration
	sf.Form.AddTextView("[white::b]Basic Configuration[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  Alias:", defaultValues.Alias, 20, nil, nil)
	sf.Form.AddInputField("  Host/IP:", defaultValues.Host, 20, nil, nil)
	sf.Form.AddInputField("  User:", defaultValues.User, 20, nil, nil)
	sf.Form.AddInputField("  Port:", defaultValues.Port, 20, nil, nil)
	sf.Form.AddInputField("  Key (comma):", defaultValues.Key, 40, nil, nil)
	sf.Form.AddInputField("  Tags (comma):", defaultValues.Tags, 30, nil, nil)

	// Connection and proxy settings
	sf.Form.AddTextView("[white::b]Connection & Proxy[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  ProxyJump:", defaultValues.ProxyJump, 40, nil, nil)
	sf.Form.AddInputField("  ProxyCommand:", defaultValues.ProxyCommand, 40, nil, nil)
	sf.Form.AddInputField("  RemoteCommand:", defaultValues.RemoteCommand, 40, nil, nil)

	// RequestTTY dropdown
	requestTTYOptions := []string{"", "yes", "no", "force", "auto"}
	requestTTYIndex := sf.findOptionIndex(requestTTYOptions, defaultValues.RequestTTY)
	sf.Form.AddDropDown("  RequestTTY:", requestTTYOptions, requestTTYIndex, nil)

	sf.Form.AddInputField("  ConnectTimeout (seconds):", defaultValues.ConnectTimeout, 10, nil, nil)
	sf.Form.AddInputField("  ConnectionAttempts:", defaultValues.ConnectionAttempts, 10, nil, nil)

	// Port forwarding
	sf.Form.AddTextView("[white::b]Port Forwarding[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  LocalForward (comma):", defaultValues.LocalForward, 40, nil, nil)
	sf.Form.AddInputField("  RemoteForward (comma):", defaultValues.RemoteForward, 40, nil, nil)
	sf.Form.AddInputField("  DynamicForward (comma):", defaultValues.DynamicForward, 40, nil, nil)

	// Authentication and key management
	sf.Form.AddTextView("[white::b]Authentication & Key Management[-]", "", 0, 1, true, false)

	// Yes/No options for dropdowns
	yesNoOptions := []string{"", "yes", "no"}

	// PubkeyAuthentication dropdown
	pubkeyIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PubkeyAuthentication)
	sf.Form.AddDropDown("  PubkeyAuthentication:", yesNoOptions, pubkeyIndex, nil)

	// PasswordAuthentication dropdown
	passwordIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PasswordAuthentication)
	sf.Form.AddDropDown("  PasswordAuthentication:", yesNoOptions, passwordIndex, nil)

	sf.Form.AddInputField("  PreferredAuthentications:", defaultValues.PreferredAuthentications, 40, nil, nil)

	// IdentitiesOnly dropdown
	identitiesOnlyIndex := sf.findOptionIndex(yesNoOptions, defaultValues.IdentitiesOnly)
	sf.Form.AddDropDown("  IdentitiesOnly:", yesNoOptions, identitiesOnlyIndex, nil)

	// AddKeysToAgent dropdown
	addKeysOptions := []string{"", "yes", "no", "ask", "confirm"}
	addKeysIndex := sf.findOptionIndex(addKeysOptions, defaultValues.AddKeysToAgent)
	sf.Form.AddDropDown("  AddKeysToAgent:", addKeysOptions, addKeysIndex, nil)

	sf.Form.AddInputField("  IdentityAgent:", defaultValues.IdentityAgent, 40, nil, nil)

	// Agent and X11 forwarding
	sf.Form.AddTextView("[white::b]Agent & X11 Forwarding[-]", "", 0, 1, true, false)

	// ForwardAgent dropdown
	forwardAgentIndex := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardAgent)
	sf.Form.AddDropDown("  ForwardAgent:", yesNoOptions, forwardAgentIndex, nil)

	// ForwardX11 dropdown
	forwardX11Index := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardX11)
	sf.Form.AddDropDown("  ForwardX11:", yesNoOptions, forwardX11Index, nil)

	// ForwardX11Trusted dropdown
	forwardX11TrustedIndex := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardX11Trusted)
	sf.Form.AddDropDown("  ForwardX11Trusted:", yesNoOptions, forwardX11TrustedIndex, nil)

	// Connection multiplexing
	sf.Form.AddTextView("[white::b]Connection Multiplexing[-]", "", 0, 1, true, false)

	// ControlMaster dropdown
	controlMasterOptions := []string{"", "yes", "no", "auto", "ask", "autoask"}
	controlMasterIndex := sf.findOptionIndex(controlMasterOptions, defaultValues.ControlMaster)
	sf.Form.AddDropDown("  ControlMaster:", controlMasterOptions, controlMasterIndex, nil)

	sf.Form.AddInputField("  ControlPath:", defaultValues.ControlPath, 40, nil, nil)
	sf.Form.AddInputField("  ControlPersist:", defaultValues.ControlPersist, 20, nil, nil)

	// Connection reliability settings
	sf.Form.AddTextView("[white::b]Connection Reliability[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  ServerAliveInterval (seconds):", defaultValues.ServerAliveInterval, 10, nil, nil)
	sf.Form.AddInputField("  ServerAliveCountMax:", defaultValues.ServerAliveCountMax, 20, nil, nil)

	// Compression dropdown
	compressionIndex := sf.findOptionIndex(yesNoOptions, defaultValues.Compression)
	sf.Form.AddDropDown("  Compression:", yesNoOptions, compressionIndex, nil)

	// TCPKeepAlive dropdown
	tcpKeepAliveIndex := sf.findOptionIndex(yesNoOptions, defaultValues.TCPKeepAlive)
	sf.Form.AddDropDown("  TCPKeepAlive:", yesNoOptions, tcpKeepAliveIndex, nil)

	// Security settings
	sf.Form.AddTextView("[white::b]Security[-]", "", 0, 1, true, false)

	// StrictHostKeyChecking dropdown
	strictHostKeyOptions := []string{"", "yes", "no", "ask", "accept-new"}
	strictHostKeyIndex := sf.findOptionIndex(strictHostKeyOptions, defaultValues.StrictHostKeyChecking)
	sf.Form.AddDropDown("  StrictHostKeyChecking:", strictHostKeyOptions, strictHostKeyIndex, nil)

	sf.Form.AddInputField("  UserKnownHostsFile:", defaultValues.UserKnownHostsFile, 40, nil, nil)
	sf.Form.AddInputField("  HostKeyAlgorithms:", defaultValues.HostKeyAlgorithms, 40, nil, nil)

	// Command execution
	sf.Form.AddTextView("[white::b]Command Execution[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  LocalCommand:", defaultValues.LocalCommand, 40, nil, nil)

	// PermitLocalCommand dropdown
	permitLocalCommandIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PermitLocalCommand)
	sf.Form.AddDropDown("  PermitLocalCommand:", yesNoOptions, permitLocalCommandIndex, nil)

	// Environment settings
	sf.Form.AddTextView("[white::b]Environment Settings[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  SendEnv (comma):", defaultValues.SendEnv, 40, nil, nil)
	sf.Form.AddInputField("  SetEnv (comma):", defaultValues.SetEnv, 40, nil, nil)

	// Debugging settings
	sf.Form.AddTextView("[white::b]Debugging[-]", "", 0, 1, true, false)

	// LogLevel dropdown
	logLevelOptions := []string{"", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"}
	logLevelIndex := sf.findOptionIndex(logLevelOptions, strings.ToUpper(defaultValues.LogLevel))
	sf.Form.AddDropDown("  LogLevel:", logLevelOptions, logLevelIndex, nil)

	// BatchMode dropdown
	batchModeIndex := sf.findOptionIndex(yesNoOptions, defaultValues.BatchMode)
	sf.Form.AddDropDown("  BatchMode:", yesNoOptions, batchModeIndex, nil)
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
	// Helper function to get text from InputField
	getFieldText := func(fieldName string) string {
		for i := 0; i < sf.Form.GetFormItemCount(); i++ {
			if field, ok := sf.Form.GetFormItem(i).(*tview.InputField); ok {
				label := strings.TrimSpace(field.GetLabel())
				if strings.HasPrefix(label, fieldName) {
					return strings.TrimSpace(field.GetText())
				}
			}
		}
		return ""
	}

	// Helper function to get selected option from DropDown
	getDropdownValue := func(fieldName string) string {
		for i := 0; i < sf.Form.GetFormItemCount(); i++ {
			if dropdown, ok := sf.Form.GetFormItem(i).(*tview.DropDown); ok {
				label := strings.TrimSpace(dropdown.GetLabel())
				if strings.HasPrefix(label, fieldName) {
					_, text := dropdown.GetCurrentOption()
					return text
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

		sf.Form.SetTitle(fmt.Sprintf("%s — [red::b]%s[-]", sf.titleForMode(), errMsg))
		sf.Form.SetBorderColor(tcell.ColorRed)
		return
	}

	sf.Form.SetTitle(sf.titleForMode())
	sf.Form.SetBorderColor(tcell.Color238)

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
