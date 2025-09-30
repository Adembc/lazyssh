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
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBar struct {
	*tview.InputField
	onSearch   func(string)
	onEscape   func()
	onEnter    func(string)
	searchTimer *time.Timer
	searchDelay time.Duration
}

func NewSearchBar() *SearchBar {
	search := &SearchBar{
		InputField:  tview.NewInputField(),
		searchDelay: 300 * time.Millisecond, // 300ms default delay
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
		// Cancel any existing timer
		if s.searchTimer != nil {
			s.searchTimer.Stop()
		}
		// Start new timer with debounced search
		s.searchTimer = time.AfterFunc(s.searchDelay, func() {
			if s.onSearch != nil {
				s.onSearch(text)
			}
		})
	})

	s.InputField.SetDoneFunc(func(key tcell.Key) {
		// Cancel any pending search when done (user pressed Enter/Esc)
		if s.searchTimer != nil {
			s.searchTimer.Stop()
			s.searchTimer = nil
		}
		if key == tcell.KeyEnter {
			if s.onEnter != nil {
				text := s.InputField.GetText()
				s.onEnter(text)
			}
		} else if key == tcell.KeyEsc {
			if s.onEscape != nil {
				s.onEscape()
			}
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

func (s *SearchBar) OnEnter(fn func(string)) *SearchBar {
	s.onEnter = fn
	return s
}

// SetSearchDelay sets the debouncing delay for search operations.
// delay specifies how long to wait after user stops typing before executing the search.
func (s *SearchBar) SetSearchDelay(delay time.Duration) *SearchBar {
	s.searchDelay = delay
	return s
}

// Stop stops any pending search timer. Should be called when the search bar is no longer needed.
func (s *SearchBar) Stop() {
	if s.searchTimer != nil {
		s.searchTimer.Stop()
		s.searchTimer = nil
	}
}
