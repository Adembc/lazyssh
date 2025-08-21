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
	"strconv"
	"strings"
	"time"

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
	title := "Add Server"
	if sf.mode == ServerFormEdit {
		title = "Edit Server"
	}

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

func (sf *ServerForm) addFormFields() {
	var defaultValues ServerFormData
	if sf.mode == ServerFormEdit && sf.original != nil {
		defaultValues = ServerFormData{
			Alias: sf.original.Alias,
			Host:  sf.original.Host,
			User:  sf.original.User,
			Port:  fmt.Sprint(sf.original.Port),
			Key:   sf.original.Key,
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
	sf.Form.AddInputField("Key:", defaultValues.Key, 40, nil, nil)
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
	if data.Alias == "" || data.Host == "" {
		return
	}

	server := sf.dataToServer(data)
	if sf.original == nil {
		server.LastSeen = time.Now()
	}
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

	return domain.Server{
		Alias: data.Alias,
		Host:  data.Host,
		User:  data.User,
		Port:  port,
		Key:   data.Key,
		Tags:  tags,
	}
}

func (sf *ServerForm) OnSave(fn func(domain.Server, *domain.Server)) *ServerForm {
	sf.onSave = fn
	return sf
}

func (sf *ServerForm) OnCancel(fn func()) *ServerForm {
	sf.onCancel = fn
	return sf
}
