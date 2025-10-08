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
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBar struct {
	*tview.InputField
	onSearch   func(string)
	onEscape   func()
	onNavigate func(direction int) // -1 for up, 1 for down
}

func NewSearchBar() *SearchBar {
	search := &SearchBar{
		InputField: tview.NewInputField(),
	}
	search.build()
	return search
}

func (s *SearchBar) build() {
	s.InputField.SetLabel(" 🔍 Search: ").
		SetFieldBackgroundColor(tcell.Color233).
		SetFieldTextColor(tcell.Color252).
		SetFieldWidth(30).
		SetBorder(true).
		SetTitle(" Search ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	s.InputField.SetChangedFunc(func(text string) {
		if s.onSearch != nil {
			s.onSearch(text)
		}
	})

	s.InputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc || key == tcell.KeyEnter {
			if s.onEscape != nil {
				s.onEscape()
			}
		}
	})

	s.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		//nolint:exhaustive // We only handle arrow keys and pass through others
		switch event.Key() {
		case tcell.KeyDown:
			if s.onNavigate != nil {
				s.onNavigate(1)
			}
			return nil
		case tcell.KeyUp:
			if s.onNavigate != nil {
				s.onNavigate(-1)
			}
			return nil
		default:
			return event
		}
	})
}

func (s *SearchBar) OnSearch(fn func(string)) *SearchBar {
	s.onSearch = fn
	return s
}

func (s *SearchBar) OnEscape(fn func()) *SearchBar {
	s.onEscape = fn
	return s
}

func (s *SearchBar) OnNavigate(fn func(direction int)) *SearchBar {
	s.onNavigate = fn
	return s
}
