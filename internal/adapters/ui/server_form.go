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

func (sf *ServerForm) addFormFields() {
	var defaultValues ServerFormData
	if sf.mode == ServerFormEdit && sf.original != nil {
		defaultValues = ServerFormData{
			Alias: sf.original.Alias,
			Host:  sf.original.Host,
			User:  sf.original.User,
			Port:  fmt.Sprint(sf.original.Port),
			Key:   strings.Join(sf.original.IdentityFiles, ", "),
			Tags:  strings.Join(sf.original.Tags, ", "),
		}
	} else {
		defaultValues = ServerFormData{
			User: "root",
			Port: "22",
			Key:  "~/.ssh/id_ed25519",
		}
	}

	sf.Form.AddInputField("Alias:", defaultValues.Alias, 20, nil, nil)
	sf.Form.AddInputField("Host/IP:", defaultValues.Host, 20, nil, nil)
	sf.Form.AddInputField("User:", defaultValues.User, 20, nil, nil)
	sf.Form.AddInputField("Port:", defaultValues.Port, 20, nil, nil)
	sf.Form.AddInputField("Key (Comma):", defaultValues.Key, 40, nil, nil)
	sf.Form.AddInputField("Tags (comma):", defaultValues.Tags, 30, nil, nil)
}

type ServerFormData struct {
	Alias string
	Host  string
	User  string
	Port  string
	Key   string
	Tags  string
}

func (sf *ServerForm) getFormData() ServerFormData {
	return ServerFormData{
		Alias: strings.TrimSpace(sf.Form.GetFormItem(0).(*tview.InputField).GetText()),
		Host:  strings.TrimSpace(sf.Form.GetFormItem(1).(*tview.InputField).GetText()),
		User:  strings.TrimSpace(sf.Form.GetFormItem(2).(*tview.InputField).GetText()),
		Port:  strings.TrimSpace(sf.Form.GetFormItem(3).(*tview.InputField).GetText()),
		Key:   strings.TrimSpace(sf.Form.GetFormItem(4).(*tview.InputField).GetText()),
		Tags:  strings.TrimSpace(sf.Form.GetFormItem(5).(*tview.InputField).GetText()),
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

	keys := []string{}
	if data.Key != "" {
		parts := strings.Split(data.Key, ",")
		for _, p := range parts {
			if k := strings.TrimSpace(p); k != "" {
				keys = append(keys, k)
			}
		}
	}
	return domain.Server{
		Alias:         data.Alias,
		Host:          data.Host,
		User:          data.User,
		Port:          port,
		IdentityFiles: keys,
		Tags:          tags,
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
