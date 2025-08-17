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

	sf.SetBorder(true).
		SetTitle(title).
		SetTitleAlign(tview.AlignLeft).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	sf.addFormFields()

	sf.AddButton("Save", sf.handleSave)
	sf.AddButton("Cancel", sf.handleCancel)
	sf.SetCancelFunc(sf.handleCancel)
}

func (sf *ServerForm) addFormFields() {
	var defaultValues ServerFormData
	if sf.mode == ServerFormEdit && sf.original != nil {
		defaultValues = ServerFormData{
			Alias:  sf.original.Alias,
			Host:   sf.original.Host,
			User:   sf.original.User,
			Port:   fmt.Sprint(sf.original.Port),
			Key:    sf.original.Key,
			Tags:   strings.Join(sf.original.Tags, ", "),
			Status: sf.original.Status,
		}
	} else {
		defaultValues = ServerFormData{
			User:   "root",
			Port:   "22",
			Key:    "~/.ssh/id_ed25519",
			Status: "online",
		}
	}

	sf.AddInputField("Alias:", defaultValues.Alias, 20, nil, nil)
	sf.AddInputField("Host/IP:", defaultValues.Host, 20, nil, nil)
	sf.AddInputField("User:", defaultValues.User, 20, nil, nil)
	sf.AddInputField("Port:", defaultValues.Port, 20, nil, nil)
	sf.AddInputField("Key:", defaultValues.Key, 40, nil, nil)
	sf.AddInputField("Tags (comma):", defaultValues.Tags, 30, nil, nil)

	statusDD := tview.NewDropDown().SetLabel("Status: ")
	statusOptions := []string{"online", "warn", "offline"}
	statusDD.SetOptions(statusOptions, nil)

	for i, opt := range statusOptions {
		if opt == defaultValues.Status {
			statusDD.SetCurrentOption(i)
			break
		}
	}
	sf.AddFormItem(statusDD)
}

type ServerFormData struct {
	Alias  string
	Host   string
	User   string
	Port   string
	Key    string
	Tags   string
	Status string
}

func (sf *ServerForm) getFormData() ServerFormData {
	return ServerFormData{
		Alias: strings.TrimSpace(sf.GetFormItem(0).(*tview.InputField).GetText()),
		Host:  strings.TrimSpace(sf.GetFormItem(1).(*tview.InputField).GetText()),
		User:  strings.TrimSpace(sf.GetFormItem(2).(*tview.InputField).GetText()),
		Port:  strings.TrimSpace(sf.GetFormItem(3).(*tview.InputField).GetText()),
		Key:   strings.TrimSpace(sf.GetFormItem(4).(*tview.InputField).GetText()),
		Tags:  strings.TrimSpace(sf.GetFormItem(5).(*tview.InputField).GetText()),
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

	_, status := sf.GetFormItem(6).(*tview.DropDown).GetCurrentOption()
	return domain.Server{
		Alias:  data.Alias,
		Host:   data.Host,
		User:   data.User,
		Port:   port,
		Key:    data.Key,
		Tags:   tags,
		Status: status,
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
