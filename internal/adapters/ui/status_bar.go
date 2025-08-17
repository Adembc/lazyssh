package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewStatusBar() *tview.TextView {
	status := tview.NewTextView().SetDynamicColors(true)
	status.SetBackgroundColor(tcell.Color235)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[white]↑↓[-] Navigate  • [white]Enter[-] SSH  • [white]a[-] Add  • [white]e[-] Edit  • [white]d[-] Delete  • [white]/[-] Search  • [white]q[-] Quit  • [white]?[-] Help")
	return status
}
