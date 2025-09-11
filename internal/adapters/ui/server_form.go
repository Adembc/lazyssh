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
			ProxyCommand:             sf.original.ProxyCommand,
			ProxyJump:                sf.original.ProxyJump,
			ForwardAgent:             sf.original.ForwardAgent,
			Compression:              sf.original.Compression,
			HostKeyAlgorithms:        sf.original.HostKeyAlgorithms,
			ServerAliveInterval:      sf.original.ServerAliveInterval,
			ServerAliveCountMax:      sf.original.ServerAliveCountMax,
			StrictHostKeyChecking:    sf.original.StrictHostKeyChecking,
			UserKnownHostsFile:       sf.original.UserKnownHostsFile,
			LogLevel:                 sf.original.LogLevel,
			PreferredAuthentications: sf.original.PreferredAuthentications,
			PasswordAuthentication:   sf.original.PasswordAuthentication,
			PubkeyAuthentication:     sf.original.PubkeyAuthentication,
			RequestTTY:               sf.original.RequestTTY,
			RemoteCommand:            sf.original.RemoteCommand,
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
	sf.Form.AddInputField("  Key (Comma):", defaultValues.Key, 40, nil, nil)
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

	// Authentication settings
	sf.Form.AddTextView("[white::b]Authentication[-]", "", 0, 1, true, false)

	// PubkeyAuthentication dropdown
	yesNoOptions := []string{"", "yes", "no"}
	pubkeyIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PubkeyAuthentication)
	sf.Form.AddDropDown("  PubkeyAuthentication:", yesNoOptions, pubkeyIndex, nil)

	// PasswordAuthentication dropdown
	passwordIndex := sf.findOptionIndex(yesNoOptions, defaultValues.PasswordAuthentication)
	sf.Form.AddDropDown("  PasswordAuthentication:", yesNoOptions, passwordIndex, nil)

	sf.Form.AddInputField("  PreferredAuthentications:", defaultValues.PreferredAuthentications, 40, nil, nil)

	// Agent and forwarding settings
	sf.Form.AddTextView("[white::b]Agent & Forwarding[-]", "", 0, 1, true, false)

	// ForwardAgent dropdown
	forwardAgentIndex := sf.findOptionIndex(yesNoOptions, defaultValues.ForwardAgent)
	sf.Form.AddDropDown("  ForwardAgent:", yesNoOptions, forwardAgentIndex, nil)

	// Connection reliability settings
	sf.Form.AddTextView("[white::b]Connection Reliability[-]", "", 0, 1, true, false)
	sf.Form.AddInputField("  ServerAliveInterval (seconds):", defaultValues.ServerAliveInterval, 10, nil, nil)
	sf.Form.AddInputField("  ServerAliveCountMax:", defaultValues.ServerAliveCountMax, 20, nil, nil)

	// Compression dropdown
	compressionIndex := sf.findOptionIndex(yesNoOptions, defaultValues.Compression)
	sf.Form.AddDropDown("  Compression:", yesNoOptions, compressionIndex, nil)

	// Security settings
	sf.Form.AddTextView("[white::b]Security[-]", "", 0, 1, true, false)

	// StrictHostKeyChecking dropdown
	strictHostKeyOptions := []string{"", "yes", "no", "ask", "accept-new"}
	strictHostKeyIndex := sf.findOptionIndex(strictHostKeyOptions, defaultValues.StrictHostKeyChecking)
	sf.Form.AddDropDown("  StrictHostKeyChecking:", strictHostKeyOptions, strictHostKeyIndex, nil)

	sf.Form.AddInputField("  UserKnownHostsFile:", defaultValues.UserKnownHostsFile, 40, nil, nil)
	sf.Form.AddInputField("  HostKeyAlgorithms:", defaultValues.HostKeyAlgorithms, 40, nil, nil)

	// Debugging settings
	sf.Form.AddTextView("[white::b]Debugging[-]", "", 0, 1, true, false)

	// LogLevel dropdown
	logLevelOptions := []string{"", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG1", "DEBUG2", "DEBUG3"}
	logLevelIndex := sf.findOptionIndex(logLevelOptions, strings.ToUpper(defaultValues.LogLevel))
	sf.Form.AddDropDown("  LogLevel:", logLevelOptions, logLevelIndex, nil)
}

type ServerFormData struct {
	Alias string
	Host  string
	User  string
	Port  string
	Key   string
	Tags  string

	// Additional SSH config fields
	ProxyCommand             string
	ProxyJump                string
	ForwardAgent             string
	Compression              string
	HostKeyAlgorithms        string
	ServerAliveInterval      string
	ServerAliveCountMax      string
	StrictHostKeyChecking    string
	UserKnownHostsFile       string
	LogLevel                 string
	PreferredAuthentications string
	PasswordAuthentication   string
	PubkeyAuthentication     string
	RequestTTY               string
	RemoteCommand            string
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
		ProxyJump:     getFieldText("ProxyJump:"),
		ProxyCommand:  getFieldText("ProxyCommand:"),
		RemoteCommand: getFieldText("RemoteCommand:"),
		RequestTTY:    getDropdownValue("RequestTTY:"),
		// Authentication settings
		PubkeyAuthentication:     getDropdownValue("PubkeyAuthentication:"),
		PasswordAuthentication:   getDropdownValue("PasswordAuthentication:"),
		PreferredAuthentications: getFieldText("PreferredAuthentications:"),
		// Agent and forwarding settings
		ForwardAgent: getDropdownValue("ForwardAgent:"),
		// Connection reliability settings
		ServerAliveInterval: getFieldText("ServerAliveInterval (seconds):"),
		ServerAliveCountMax: getFieldText("ServerAliveCountMax:"),
		Compression:         getDropdownValue("Compression:"),
		// Security settings
		StrictHostKeyChecking: getDropdownValue("StrictHostKeyChecking:"),
		UserKnownHostsFile:    getFieldText("UserKnownHostsFile:"),
		HostKeyAlgorithms:     getFieldText("HostKeyAlgorithms:"),
		// Debugging settings
		LogLevel: strings.ToLower(getDropdownValue("LogLevel:")),
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
	return domain.Server{
		Alias:                    data.Alias,
		Host:                     data.Host,
		User:                     data.User,
		Port:                     port,
		IdentityFiles:            keys,
		Tags:                     tags,
		ProxyCommand:             data.ProxyCommand,
		ProxyJump:                data.ProxyJump,
		ForwardAgent:             data.ForwardAgent,
		Compression:              data.Compression,
		HostKeyAlgorithms:        data.HostKeyAlgorithms,
		ServerAliveInterval:      data.ServerAliveInterval,
		ServerAliveCountMax:      data.ServerAliveCountMax,
		StrictHostKeyChecking:    data.StrictHostKeyChecking,
		UserKnownHostsFile:       data.UserKnownHostsFile,
		LogLevel:                 data.LogLevel,
		PreferredAuthentications: data.PreferredAuthentications,
		PasswordAuthentication:   data.PasswordAuthentication,
		PubkeyAuthentication:     data.PubkeyAuthentication,
		RequestTTY:               data.RequestTTY,
		RemoteCommand:            data.RemoteCommand,
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
