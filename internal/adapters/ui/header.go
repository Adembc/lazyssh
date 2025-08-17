package ui

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AppHeader struct {
	*tview.Flex
	version   string
	buildTime string
	gitCommit string
	repoURL   string
}

func NewAppHeader(version, gitCommit, buildTime, repoURL string) *AppHeader {
	header := &AppHeader{
		Flex:      tview.NewFlex(),
		version:   version,
		repoURL:   repoURL,
		buildTime: buildTime,
		gitCommit: gitCommit,
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

	commit := shortCommit(h.gitCommit)

	// Build tag-like chips for version, commit, and build time
	versionTag := makeTag(h.version, "#22C55E") // green
	commitTag := ""
	if commit != "" {
		commitTag = makeTag(commit, "#A78BFA") // violet
	}
	timeTag := makeTag(formatBuildTime(h.buildTime), "#3B82F6") // blue

	text := versionTag
	if commitTag != "" {
		text += "  " + commitTag
	}
	text += "  " + timeTag

	center.SetText(text)
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

// shortCommit returns first 7 chars of commit if it looks valid; otherwise empty string.
func shortCommit(c string) string {
	c = strings.TrimSpace(c)
	if c == "" || c == "unknown" || c == "(devel)" {
		return ""
	}
	if len(c) > 7 {
		return c[:7]
	}
	return c
}

// formatBuildTime tries to parse common time formats and returns a concise human-readable string.
func formatBuildTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC1123,
		time.RFC1123Z,
	}
	var t time.Time
	var err error
	for _, l := range layouts {
		t, err = time.Parse(l, s)
		if err == nil {
			return t.Format("Mon, 02 Jan 2006 15:04")
		}
	}

	return s
}

// makeTag returns a rectangular-looking colored chip for the given text.
func makeTag(text, bg string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return "[black:" + bg + "::b]  " + text + "  [-]"
}
