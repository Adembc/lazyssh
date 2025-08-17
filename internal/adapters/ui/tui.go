package ui

import (
	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"time"
)

type tui struct {
	app           *tview.Application
	serverService ports.ServerService

	header     *AppHeader
	searchBar  *SearchBar
	hintBar    *tview.TextView
	serverList *ServerList
	details    *ServerDetails
	statusBar  *tview.TextView

	root    *tview.Flex
	left    *tview.Flex
	content *tview.Flex

	searchVisible bool
}

func NewTUI(ss ports.ServerService) *tui {
	return &tui{
		app:           tview.NewApplication(),
		serverService: ss,
	}

}

func (t *tui) Run() error {

	t.app.EnableMouse(true)
	t.initializeTheme().buildComponents().buildLayout().bindEvents().loadInitialData().loadSplashScreen()

	return t.app.Run()
}
func (t *tui) initializeTheme() *tui {
	tview.Styles.PrimitiveBackgroundColor = tcell.Color232
	tview.Styles.ContrastBackgroundColor = tcell.Color235
	tview.Styles.BorderColor = tcell.Color238
	tview.Styles.TitleColor = tcell.Color250
	tview.Styles.PrimaryTextColor = tcell.Color252
	tview.Styles.TertiaryTextColor = tcell.Color245
	tview.Styles.SecondaryTextColor = tcell.Color245
	tview.Styles.GraphicsColor = tcell.Color238
	return t
}
func (t *tui) buildComponents() *tui {
	t.header = NewAppHeader(AppVersion, RepoURL)
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.hideSearchBar)
	t.hintBar = NewHintBar()
	t.serverList = NewServerList().
		OnSelectionChange(t.handleServerSelectionChange)
	t.details = NewServerDetails()
	t.statusBar = NewStatusBar()
	return t
}
func (t *tui) buildLayout() *tui {

	t.left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.hintBar, 1, 0, false).
		AddItem(t.serverList, 0, 1, true)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.details, 0, 1, false)

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(t.left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 2, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
	return t
}
func (t *tui) bindEvents() *tui {
	t.root.SetInputCapture(t.handleGlobalKeys)
	return t
}
func (t *tui) loadInitialData() *tui {
	servers, _ := t.serverService.ListServers("")
	t.serverList.UpdateServers(servers)

	return t
}

func (t *tui) loadSplashScreen() *tui {
	splash, stop := buildSplash(t.app)
	t.app.SetRoot(splash, true)
	time.AfterFunc(SplashScreenDuration, func() {
		stop()
		t.app.QueueUpdateDraw(func() {
			t.app.SetRoot(t.root, true)
		})
	})
	return t

}
