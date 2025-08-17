package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type SearchBar struct {
	*tview.InputField
	onSearch func(string)
	onEscape func()
}

func NewSearchBar() *SearchBar {
	search := &SearchBar{
		InputField: tview.NewInputField(),
	}
	search.build()
	return search
}

func (s *SearchBar) build() {
	s.SetLabel(" 🔍 Search: ").
		SetFieldBackgroundColor(tcell.Color233).
		SetFieldTextColor(tcell.Color252).
		SetFieldWidth(30).
		SetBorder(true).
		SetTitle("Search").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	s.SetChangedFunc(func(text string) {
		if s.onSearch != nil {
			s.onSearch(text)
		}
	})

	s.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEsc || key == tcell.KeyEnter {
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
