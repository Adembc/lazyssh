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

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SSHKeysList struct {
	*tview.List
	keys              []domain.SSHKey
	onSelectionChange func(domain.SSHKey)
}

func NewSSHKeysList() *SSHKeysList {
	list := &SSHKeysList{
		List: tview.NewList(),
	}
	list.build()
	return list
}

func (kl *SSHKeysList) build() {
	kl.List.ShowSecondaryText(false)
	kl.List.SetBorder(true).
		SetTitle(" SSH Keys ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)
	kl.List.
		SetSelectedBackgroundColor(tcell.Color24).
		SetSelectedTextColor(tcell.Color255).
		SetHighlightFullLine(true)

	kl.List.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(kl.keys) && kl.onSelectionChange != nil {
			kl.onSelectionChange(kl.keys[index])
		}
	})
}

func (kl *SSHKeysList) UpdateKeys(keys []domain.SSHKey) {
	kl.keys = keys
	kl.List.Clear()

	for i := range keys {
		primary := formatSSHKeyLine(keys[i])
		idx := i
		kl.List.AddItem(primary, "", 0, func() {
			// No selection action on Enter for keys
			_ = idx
		})
	}

	if kl.List.GetItemCount() > 0 {
		kl.List.SetCurrentItem(0)
		if kl.onSelectionChange != nil {
			kl.onSelectionChange(kl.keys[0])
		}
	}
}

func (kl *SSHKeysList) GetSelectedKey() (domain.SSHKey, bool) {
	idx := kl.List.GetCurrentItem()
	if idx >= 0 && idx < len(kl.keys) {
		return kl.keys[idx], true
	}
	return domain.SSHKey{}, false
}

func (kl *SSHKeysList) OnSelectionChange(fn func(key domain.SSHKey)) *SSHKeysList {
	kl.onSelectionChange = fn
	return kl
}

func (kl *SSHKeysList) GetCurrentItem() int {
	return kl.List.GetCurrentItem()
}

func (kl *SSHKeysList) GetItemCount() int {
	return kl.List.GetItemCount()
}

func (kl *SSHKeysList) SetCurrentItem(index int) {
	kl.List.SetCurrentItem(index)
}

func formatSSHKeyLine(key domain.SSHKey) string {
	// Format: "[indicator] name (type:size)"
	indicator := " "
	if key.LoadedInAgent {
		indicator = "[green]●[-]" // Green dot for loaded keys
	}

	typeInfo := key.Type
	if key.Size > 0 {
		typeInfo = fmt.Sprintf("%s:%d", key.Type, key.Size)
	}

	flags := ""
	if key.IsEncrypted {
		flags += "[yellow]🔒[-] "
	}
	if !key.HasPublicKey {
		flags += "[red]⛅[-] "
	}

	return fmt.Sprintf("%s %s%s [dim](%s)[-]", indicator, flags, key.Name, typeInfo)
}
