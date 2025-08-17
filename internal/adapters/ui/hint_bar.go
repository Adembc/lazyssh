package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewHintBar() *tview.TextView {
	hint := tview.NewTextView().SetDynamicColors(true)
	hint.SetBackgroundColor(tcell.Color233)
	hint.SetText("[#BBBBBB]Press [::b]/[-:-:b] to search…  •  ↑↓ Navigate  •  Enter SSH  •  a Add  •  e Edit  •  d Delete  •  ? Help[-]")
	return hint
}
