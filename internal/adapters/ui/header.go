package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"time"
)

type AppHeader struct {
	*tview.Flex
	version string
	repoURL string
}

func NewAppHeader(version, repoURL string) *AppHeader {
	header := &AppHeader{
		Flex:    tview.NewFlex(),
		version: version,
		repoURL: repoURL,
	}
	header.build()
	return header
}

func (h *AppHeader) build() {
	headerBg := tcell.Color234

	left := h.buildLeftSection(headerBg)
	center := h.buildCenterSection(headerBg)
	right := h.buildRightSection(headerBg)

	headerBar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 1, false).
		AddItem(center, 0, 1, false).
		AddItem(right, 0, 1, false)

	separator := h.createSeparator()

	h.SetDirection(tview.FlexRow).
		AddItem(headerBar, 1, 0, false).
		AddItem(separator, 1, 0, false)
}

func (h *AppHeader) buildLeftSection(bg tcell.Color) *tview.TextView {
	left := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	left.SetBackgroundColor(bg)
	stylizedName := "🚀 [#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
	left.SetText(stylizedName)
	return left
}

func (h *AppHeader) buildCenterSection(bg tcell.Color) *tview.TextView {
	center := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	center.SetBackgroundColor(bg)
	center.SetText("[black:#2ECC71::b]  " + h.version + "  [-]  [black:#3B82F6::b]  TUI  [-]")
	return center
}

func (h *AppHeader) buildRightSection(bg tcell.Color) *tview.TextView {
	right := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	right.SetBackgroundColor(bg)
	currentTime := time.Now().Format("Mon, 02 Jan 2006 15:04")
	right.SetText("[#55AAFF::u]🔗 " + h.repoURL + "[-]  [#AAAAAA]• " + currentTime + "[-]")
	return right
}

func (h *AppHeader) createSeparator() *tview.TextView {
	separator := tview.NewTextView().SetDynamicColors(true)
	separator.SetBackgroundColor(tcell.Color235)
	separator.SetText("[#444444]" + strings.Repeat("─", 200) + "[-]")
	return separator
}
