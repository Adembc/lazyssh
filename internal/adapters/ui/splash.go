package ui

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// buildSplash creates a splash screen with a spinner; returns primitive and a stop function
func buildSplash(app *tview.Application) (tview.Primitive, func()) {
	// Brand styling
	title := "[#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
	tagline := "[#AAAAAA]Fast server picker TUI • Loading your servers…[-]"
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	text := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	text.SetBorder(true).
		SetTitle(" 🚀  lazyssh ").
		SetBorderColor(tcell.Color238).
		SetTitleColor(tcell.Color250)

	// layout to center the card
	box := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(text, 9, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	container := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(box, 0, 2, false).
		AddItem(tview.NewBox(), 0, 1, false)

	stop := make(chan struct{})
	go func() {
		i := 0
		for {
			select {
			case <-stop:
				return
			case <-time.After(90 * time.Millisecond):
				frame := spinnerFrames[i%len(spinnerFrames)]

				width := 20
				filled := i % (width + 1)
				prog := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

				// rotate tips every ~12 frames
				i++
				app.QueueUpdateDraw(func() {
					text.SetText(
						"\n" +
							"[::b]" + title + "[-]\n" +
							"\n" + frame + "  " + tagline + "\n" +
							"\n[#55D7FF]" + prog,
					)
				})
			}
		}
	}()

	stopFunc := func() { close(stop) }
	return container, stopFunc
}
