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
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// EditKeyComment is a modal dialog for editing SSH key comments.
type EditKeyComment struct {
	*tview.Flex
	app            *tview.Application
	form           *tview.Form
	infoText       *tview.TextView
	keyPath        string
	keyName        string
	initialComment string
	onSave         func(comment string)
	onCancel       func()
}

// NewEditKeyComment creates a new comment edit modal.
func NewEditKeyComment(app *tview.Application) *EditKeyComment {
	form := tview.NewForm()
	form.SetBorderPadding(1, 1, 2, 2)

	infoText := tview.NewTextView()
	infoText.SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetBorderPadding(1, 1, 2, 2)

	edit := &EditKeyComment{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		app:      app,
		form:     form,
		infoText: infoText,
	}

	edit.AddItem(edit.infoText, 0, 1, false).
		AddItem(edit.form, 0, 2, true)

	edit.SetBorder(true).
		SetTitle(" Edit SSH Key Comment ").
		SetTitleAlign(tview.AlignLeft)

	return edit
}

// SetKey sets the key name, path, and initial comment for editing.
func (e *EditKeyComment) SetKey(name, path, comment string) *EditKeyComment {
	e.keyName = name
	e.keyPath = path
	e.initialComment = comment
	return e
}

// OnSave sets the callback for when the user saves the comment.
func (e *EditKeyComment) OnSave(fn func(comment string)) *EditKeyComment {
	e.onSave = fn
	return e
}

// OnCancel sets the callback for when the user cancels.
func (e *EditKeyComment) OnCancel(fn func()) *EditKeyComment {
	e.onCancel = fn
	return e
}

// Show displays the modal dialog.
func (e *EditKeyComment) Show() error {
	// Build info text
	infoText := fmt.Sprintf("Editing comment for SSH key:\n\n[yellow]%s[-]\n[dim]%s[-]", e.keyName, e.keyPath)
	e.infoText.SetText(infoText)

	// Clear and rebuild form
	e.form.Clear(true)

	// Add comment input field with validation
	e.form.AddInputField("Comment:", e.initialComment, 0,
		func(textToCheck string, lastChar rune) bool {
			// Limit to 220 characters
			return len(textToCheck) <= 220
		}, nil).
		SetLabelColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.Color236)

	// Add buttons
	e.form.AddButton("Save", func() {
		e.save()
	})
	e.form.AddButton("Cancel", func() {
		e.cancel()
	})

	// Set up key bindings
	e.form.SetInputCapture(e.handleInput)

	// Focus the form
	e.app.SetFocus(e.form)

	// Set as modal
	e.app.SetRoot(e, true)

	return nil
}

func (e *EditKeyComment) handleInput(event *tcell.EventKey) *tcell.EventKey {
	//nolint:exhaustive // We only handle specific keys, default handles the rest
	switch event.Key() {
	case tcell.KeyEscape, tcell.KeyCtrlC:
		e.cancel()
		return nil
	case tcell.KeyCtrlS:
		e.save()
		return nil
	default:
		// Let the form handle all other input
		return event
	}
}

func (e *EditKeyComment) save() {
	// Get the comment from the input field (index 0)
	comment := e.form.GetFormItem(0).(*tview.InputField).GetText()

	// Trim whitespace
	comment = strings.TrimSpace(comment)

	// Enforce max length
	if len(comment) > 220 {
		comment = comment[:220]
	}

	// Call the save callback
	if e.onSave != nil {
		e.onSave(comment)
	}

	e.cancel()
}

func (e *EditKeyComment) cancel() {
	if e.onCancel != nil {
		e.onCancel()
	}
}
