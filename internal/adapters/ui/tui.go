package ui

import (
	"fmt"
	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
	"time"
)

// splash screen with spinner; returns primitive and a stop function
func buildSplash(app *tview.Application) (tview.Primitive, func()) {
	// Brand styling
	title := "[#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
	tagline := "[#AAAAAA]Fast server picker TUI • Loading your servers…[-]"
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	// Helpful rotating tips
	tips := []string{
		"Press [::b]/[-] to search instantly",
		"Use [::b]↑/↓[-] to navigate",
		"Hit [::b]Enter[-] to open a connection (mock)",
		"Press [::b]?[-] for help",
		"Press [::b]q[-] to quit",
	}

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
				// simple progress bar animation (width 20)
				width := 20
				filled := i % (width + 1)
				prog := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
				// rotate tips every ~12 frames
				tip := tips[(i/12)%len(tips)]
				i++
				app.QueueUpdateDraw(func() {
					text.SetText(
						"\n" +
							"[::b]" + title + "[-]\n" +
							"\n" + frame + "  " + tagline + "\n" +
							"\n[#55D7FF]" + prog + "[-]\n" +
							"\n[#777777]" + tip + "[-]\n" +
							"\n[#555555]v" + AppVersion + " • " + RepoURL + "[-]",
					)
				})
			}
		}
	}()

	stopFunc := func() { close(stop) }
	return container, stopFunc
}

func showHelpModal(app *tview.Application, root tview.Primitive) {
	text := "Keyboard shortcuts:\n\n" +
		"  ↑/↓          Navigate\n" +
		"  Enter        SSH connect (mock)\n" +
		"  a            Add server (mock)\n" +
		"  e            Edit server (mock)\n" +
		"  d            Delete entry (mock)\n" +
		"  /            Focus search\n" +
		"  q            Quit\n" +
		"  ?            Help\n"

	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(root, true)
		})

	app.SetRoot(modal, true)
}

func showConnectModal(app *tview.Application, root tview.Primitive, s domain.Server) {
	msg := fmt.Sprintf("SSH to %s (%s@%s:%d)\n\nThis is a mock action.", s.Alias, s.User, s.Host, s.Port)
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(root, true)
		})
	app.SetRoot(modal, true)
}

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
	splash, stop := buildSplash(t.app)
	t.app.SetRoot(splash, true)
	time.AfterFunc(1*time.Second, func() {
		stop()
		t.app.QueueUpdateDraw(func() {
			t.app.SetRoot(t.root, true)
		})
	})

	t.app.EnableMouse(true)
	t.initializeTheme().buildComponents().buildLayout().bindEvents().loadInitialData()

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
		OnSearch(t.handleSearch).
		OnEscape(t.hideSearchBar)
	t.hintBar = NewHintBar()
	t.serverList = NewServerList().
		OnSelection(t.handleServerConnect).
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

//func (t *tui) Run() error {
//	// Global theme (non-invasive)
//	tview.Styles.PrimitiveBackgroundColor = tcell.Color232 // near-black
//	tview.Styles.ContrastBackgroundColor = tcell.Color235  // dark gray
//	tview.Styles.BorderColor = tcell.Color238
//	tview.Styles.TitleColor = tcell.Color250
//	tview.Styles.PrimaryTextColor = tcell.Color252
//	tview.Styles.TertiaryTextColor = tcell.Color245
//	tview.Styles.SecondaryTextColor = tcell.Color245
//	tview.Styles.GraphicsColor = tcell.Color238
//
//	servers, err := t.serverService.ListServers()
//	if err != nil {
//		return err
//	}
//	// Sort by alias for deterministic order
//	sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
//
//	// Header: enhanced two-row layout (bar + separator)
//	version := AppVersion
//	repoURL := RepoURL
//
//	// Slightly lighter than body for a distinct bar
//	headerBg := tcell.Color234 // dark gray tint
//	// Left: App icon + stylized name
//	headerLeft := tview.NewTextView().
//		SetDynamicColors(true).
//		SetTextAlign(tview.AlignLeft)
//	headerLeft.SetBackgroundColor(headerBg)
//	stylizedName := "🚀 [#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
//	headerLeft.SetText(stylizedName)
//
//	// Center: badges (version + type)
//	headerCenter := tview.NewTextView().
//		SetDynamicColors(true).
//		SetTextAlign(tview.AlignCenter)
//	headerCenter.SetBackgroundColor(headerBg)
//	// Render version as a pill/badge and add a secondary badge
//	headerCenter.SetText("[black:#2ECC71::b]  " + version + "  [-]  [black:#3B82F6::b]  TUI  [-]")
//
//	// Right: repo URL + current date/time
//	headerRight := tview.NewTextView().
//		SetDynamicColors(true).
//		SetTextAlign(tview.AlignRight)
//	headerRight.SetBackgroundColor(headerBg)
//	currentTime := time.Now().Format("Mon, 02 Jan 2006 15:04")
//	headerRight.SetText("[#55AAFF::u]🔗 " + repoURL + "[-]  [#AAAAAA]• " + currentTime + "[-]")
//
//	// Row 1: the main header bar with three columns
//	headerBar := tview.NewFlex().SetDirection(tview.FlexColumn).
//		AddItem(headerLeft, 0, 1, false).
//		AddItem(headerCenter, 0, 1, false).
//		AddItem(headerRight, 0, 1, false)
//
//	// Row 2: a thin separator line
//	separator := tview.NewTextView().SetDynamicColors(true)
//	separator.SetBackgroundColor(tcell.Color235)
//	separator.SetText("[#444444]" + strings.Repeat("─", 200) + "[-]")
//
//	// Combine into a two-row header
//	header := tview.NewFlex().SetDirection(tview.FlexRow).
//		AddItem(headerBar, 1, 0, false).
//		AddItem(separator, 1, 0, false)
//
//	// Tab bar removed per request
//
//	// Left panel: Search is hidden by default; a hint bar is shown until '/' is pressed
//	search := tview.NewInputField().
//		SetLabel(" 🔍 Search: ").
//		SetFieldWidth(30)
//	// When visible, give it a border and title for better UI
//	search.SetBorder(true).SetTitle("Search")
//	search.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
//	search.SetFieldBackgroundColor(tcell.Color233).SetFieldTextColor(tcell.Color252)
//
//	serverList := tview.NewList().ShowSecondaryText(false)
//	serverList.SetBorder(true)
//	serverList.SetTitle("Servers")
//	serverList.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
//	serverList.SetSelectedBackgroundColor(tcell.Color24).SetSelectedTextColor(tcell.Color255)
//	serverList.SetHighlightFullLine(true)
//
//	// Hint bar shown when search is hidden
//	hintBar := tview.NewTextView().SetDynamicColors(true)
//	hintBar.SetBackgroundColor(tcell.Color233)
//	hintBar.SetText("[#BBBBBB]Press [::b]/[-:-:b] to search…  •  ↑↓ Navigate  •  Enter SSH  •  a Add  •  e Edit  •  d Delete  •  ? Help[-]")
//
//	// Right panel: Details
//	details := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
//	details.SetBorder(true).SetTitle("Details")
//	details.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
//
//	// Status bar
//	status := tview.NewTextView().SetDynamicColors(true)
//	statusBg := tcell.Color235
//	status.SetBackgroundColor(statusBg)
//	status.SetTextAlign(tview.AlignCenter)
//	status.SetText("[white]↑↓[-] Navigate  • [white]Enter[-] SSH  • [white]a[-] Add  • [white]e[-] Edit  • [white]d[-] Delete  • [white]/[-] Search  • [white]q[-] Quit  • [white]?[-] Help")
//
//	// Define updateDetails before populate; declare var for closure use
//	var updateDetails func(server domain.Server)
//	// Root container placeholder (assigned later)
//	var root *tview.Flex
//
//	// Populate list with filter logic
//	filtered := servers
//	populate := func() {
//		serverList.Clear()
//		for i := range filtered {
//			p, s2 := formatServerLine(filtered[i])
//			idx := i
//			serverList.AddItem(p, s2, 0, func() {
//				if updateDetails != nil {
//					updateDetails(filtered[idx])
//				}
//			})
//		}
//		if serverList.GetItemCount() > 0 {
//			serverList.SetCurrentItem(0)
//			if updateDetails != nil {
//				updateDetails(filtered[0])
//			}
//		} else {
//			details.SetText("No servers match the current filter.")
//		}
//	}
//
//	// Will be defined later to refresh according to current search
//	var reapplyFilter func()
//
//	// Delete currently selected server from memory and refresh
//	deleteSelected := func() {
//		idx := serverList.GetCurrentItem()
//		if idx < 0 || idx >= len(filtered) {
//			return
//		}
//		sel := filtered[idx]
//		// remove from servers slice
//		newServers := make([]domain.Server, 0, len(servers))
//		for _, s := range servers {
//			if !(s.Alias == sel.Alias && s.Host == sel.Host) {
//				newServers = append(newServers, s)
//			}
//		}
//		servers = newServers
//		reapplyFilter()
//	}
//
//	// Now define updateDetails
//	updateDetails = func(s domain.Server) {
//		text := fmt.Sprintf("[::b]%s[-]\n\nHost: [white]%s[-]\nUser: [white]%s[-]\nPort: [white]%d[-]\nKey:  [white]%s[-]\nTags: [white]%s[-]\nStatus: %s\nLast: %s\n\n[::b]Commands:[-]\n  Enter: SSH connect\n  a: Add new server\n  e: Edit entry\n  d: Delete entry",
//			s.Alias, s.Host, s.User, s.Port, s.Key, joinTags(s.Tags), statusIcon(s.Status), s.LastSeen.Format("2006-01-02 15:04"))
//		details.SetText(text)
//	}
//
//	// Helper to re-apply current filter and repopulate
//	reapplyFilter = func() {
//		q := strings.TrimSpace(strings.ToLower(search.GetText()))
//		if q == "" {
//			filtered = servers
//			populate()
//			return
//		}
//		var out []domain.Server
//		for _, s := range servers {
//			hay := strings.ToLower(strings.Join([]string{
//				s.Alias,
//				s.Host,
//				s.User,
//				fmt.Sprint(s.Port),
//				strings.Join(s.Tags, ","),
//				s.Key,
//			}, " "))
//			if strings.Contains(hay, q) {
//				out = append(out, s)
//			}
//		}
//		filtered = out
//		populate()
//	}
//
//	// Add new server form
//	addServer := func() {
//		aliasField := tview.NewInputField().SetLabel("Alias: ")
//		hostField := tview.NewInputField().SetLabel("Host/IP: ")
//		userField := tview.NewInputField().SetLabel("User: ").SetText("root")
//		portField := tview.NewInputField().SetLabel("Port: ").SetText("22")
//		keyField := tview.NewInputField().SetLabel("Key: ").SetText("~/.ssh/id_ed25519")
//		tagsField := tview.NewInputField().SetLabel("Tags (comma): ")
//		statusDD := tview.NewDropDown().SetLabel("Status: ")
//		statusOptions := []string{"online", "warn", "offline"}
//		statusDD.SetOptions(statusOptions, nil)
//		statusDD.SetCurrentOption(0)
//
//		form := tview.NewForm().
//			AddFormItem(aliasField).
//			AddFormItem(hostField).
//			AddFormItem(userField).
//			AddFormItem(portField).
//			AddFormItem(keyField).
//			AddFormItem(tagsField).
//			AddFormItem(statusDD)
//		form.SetBorder(true).SetTitle("Add Server").SetTitleAlign(tview.AlignLeft)
//		form.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
//		form.AddButton("Save", func() {
//			alias := strings.TrimSpace(aliasField.GetText())
//			host := strings.TrimSpace(hostField.GetText())
//			if alias == "" || host == "" {
//				return
//			}
//			user := strings.TrimSpace(userField.GetText())
//			p := 22
//			if v := strings.TrimSpace(portField.GetText()); v != "" {
//				if n, err := strconv.Atoi(v); err == nil && n > 0 {
//					p = n
//				}
//			}
//			key := strings.TrimSpace(keyField.GetText())
//			tags := []string{}
//			if ts := strings.TrimSpace(tagsField.GetText()); ts != "" {
//				for _, t := range strings.Split(ts, ",") {
//					if s := strings.TrimSpace(t); s != "" {
//						tags = append(tags, s)
//					}
//				}
//			}
//			_, status := statusDD.GetCurrentOption()
//			newS := domain.Server{Alias: alias, Host: host, User: user, Port: p, Key: key, Tags: tags, Status: status, LastSeen: time.Now()}
//			servers = append(servers, newS)
//			sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
//			reapplyFilter()
//			// select the new item if it is visible
//			for i := range filtered {
//				if filtered[i].Alias == newS.Alias && filtered[i].Host == newS.Host {
//					serverList.SetCurrentItem(i)
//					updateDetails(filtered[i])
//					break
//				}
//			}
//			t.app.SetRoot(root, true)
//		})
//		form.AddButton("Cancel", func() { t.app.SetRoot(root, true) })
//		form.SetCancelFunc(func() { t.app.SetRoot(root, true) })
//		t.app.SetRoot(form, true)
//	}
//
//	// Edit currently selected server
//	editSelected := func() {
//		idx := serverList.GetCurrentItem()
//		if idx < 0 || idx >= len(filtered) {
//			return
//		}
//		orig := filtered[idx]
//		aliasField := tview.NewInputField().SetLabel("Alias: ").SetText(orig.Alias)
//		hostField := tview.NewInputField().SetLabel("Host/IP: ").SetText(orig.Host)
//		userField := tview.NewInputField().SetLabel("User: ").SetText(orig.User)
//		portField := tview.NewInputField().SetLabel("Port: ").SetText(fmt.Sprint(orig.Port))
//		keyField := tview.NewInputField().SetLabel("Key: ").SetText(orig.Key)
//		tagsField := tview.NewInputField().SetLabel("Tags (comma): ").SetText(strings.Join(orig.Tags, ", "))
//		statusDD := tview.NewDropDown().SetLabel("Status: ")
//		statusOptions := []string{"online", "warn", "offline"}
//		statusDD.SetOptions(statusOptions, nil)
//		// set current option to existing status
//		curIdx := 0
//		for i, o := range statusOptions {
//			if o == orig.Status {
//				curIdx = i
//				break
//			}
//		}
//		statusDD.SetCurrentOption(curIdx)
//
//		form := tview.NewForm().
//			AddFormItem(aliasField).
//			AddFormItem(hostField).
//			AddFormItem(userField).
//			AddFormItem(portField).
//			AddFormItem(keyField).
//			AddFormItem(tagsField).
//			AddFormItem(statusDD)
//		form.SetBorder(true).SetTitle("Edit Server").SetTitleAlign(tview.AlignLeft)
//		form.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
//		form.AddButton("Save", func() {
//			alias := strings.TrimSpace(aliasField.GetText())
//			host := strings.TrimSpace(hostField.GetText())
//			if alias == "" || host == "" {
//				return
//			}
//			user := strings.TrimSpace(userField.GetText())
//			p := orig.Port
//			if v := strings.TrimSpace(portField.GetText()); v != "" {
//				if n, err := strconv.Atoi(v); err == nil && n > 0 {
//					p = n
//				}
//			}
//			key := strings.TrimSpace(keyField.GetText())
//			tags := []string{}
//			if ts := strings.TrimSpace(tagsField.GetText()); ts != "" {
//				for _, t := range strings.Split(ts, ",") {
//					if s := strings.TrimSpace(t); s != "" {
//						tags = append(tags, s)
//					}
//				}
//			}
//			_, status := statusDD.GetCurrentOption()
//			updated := domain.Server{Alias: alias, Host: host, User: user, Port: p, Key: key, Tags: tags, Status: status, LastSeen: orig.LastSeen}
//			// replace matching item in servers by original key alias+host
//			found := false
//			for i := range servers {
//				if servers[i].Alias == orig.Alias && servers[i].Host == orig.Host {
//					servers[i] = updated
//					found = true
//					break
//				}
//			}
//			if !found {
//				servers = append(servers, updated)
//			}
//			sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
//			reapplyFilter()
//			// select the updated item if visible
//			for i := range filtered {
//				if filtered[i].Alias == updated.Alias && filtered[i].Host == updated.Host {
//					serverList.SetCurrentItem(i)
//					updateDetails(filtered[i])
//					break
//				}
//			}
//			t.app.SetRoot(root, true)
//		})
//		form.AddButton("Cancel", func() { t.app.SetRoot(root, true) })
//		form.SetCancelFunc(func() { t.app.SetRoot(root, true) })
//		t.app.SetRoot(form, true)
//	}
//
//	filter := func(q string) {
//		q = strings.TrimSpace(strings.ToLower(q))
//		if q == "" {
//			filtered = servers
//			populate()
//			return
//		}
//		var out []domain.Server
//		for _, s := range servers {
//			hay := strings.ToLower(strings.Join([]string{
//				s.Alias,
//				s.Host,
//				s.User,
//				fmt.Sprint(s.Port),
//				strings.Join(s.Tags, ","),
//				s.Key,
//			}, " "))
//			if strings.Contains(hay, q) {
//				out = append(out, s)
//			}
//		}
//		filtered = out
//		populate()
//	}
//
//	// Declare togglers; define after left is created
//	var showSearchBar func()
//	var hideSearchBar func()
//
//	search.SetChangedFunc(func(text string) { filter(text) })
//	search.SetDoneFunc(func(key tcell.Key) {
//		// Return focus to list on Escape or Enter and hide the search bar
//		if key == tcell.KeyEsc || key == tcell.KeyEnter {
//			if hideSearchBar != nil {
//				hideSearchBar()
//			} else {
//				t.app.SetFocus(serverList)
//			}
//		}
//	})
//
//	// Initial populate
//	populate()
//
//	// Layout main content area
//	var left *tview.Flex
//	left = tview.NewFlex().SetDirection(tview.FlexRow).
//		AddItem(hintBar, 1, 0, false).
//		AddItem(serverList, 0, 1, true)
//
//	right := tview.NewFlex().SetDirection(tview.FlexRow).
//		AddItem(details, 0, 1, false)
//
//	content := tview.NewFlex().SetDirection(tview.FlexColumn).
//		AddItem(left, 0, 3, true).
//		AddItem(right, 0, 2, false)
//
//	// Define togglers now that 'left' exists
//	showSearchBar = func() {
//		left.Clear()
//		left.AddItem(search, 3, 0, true)
//		left.AddItem(serverList, 0, 1, false)
//		t.app.SetFocus(search)
//	}
//	hideSearchBar = func() {
//		left.Clear()
//		left.AddItem(hintBar, 1, 0, false)
//		left.AddItem(serverList, 0, 1, true)
//		t.app.SetFocus(serverList)
//	}
//
//	// Root layout: header, content, status
//	root = tview.NewFlex().SetDirection(tview.FlexRow).
//		AddItem(header, 2, 0, false).
//		AddItem(content, 0, 1, true).
//		AddItem(status, 1, 0, false)
//
//	// Keybindings
//	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
//		// If the search input has focus, do not trigger global shortcuts.
//		if t.app.GetFocus() == search {
//			return event
//		}
//		switch event.Rune() {
//		case 'q':
//			t.app.Stop()
//			return nil
//		case '/':
//			if showSearchBar != nil {
//				showSearchBar()
//			} else {
//				t.app.SetFocus(search)
//			}
//			return nil
//		case 'a':
//			addServer()
//			return nil
//		case 'e':
//			editSelected()
//			return nil
//		case 'd':
//			deleteSelected()
//			return nil
//		case '?':
//			showHelpModal(t.app, root)
//			return nil
//		}
//		// Handle Enter at root level as connect for current selection
//		if event.Key() == tcell.KeyEnter {
//			idx := serverList.GetCurrentItem()
//			if idx >= 0 && idx < len(filtered) {
//				showConnectModal(t.app, root, filtered[idx])
//				return nil
//			}
//		}
//		return event
//	})
//
//	// Also handle list-selected changes to update details
//	serverList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
//		if index >= 0 && index < len(filtered) {
//			updateDetails(filtered[index])
//		}
//	})
//
//	// Always show the splash screen for 1 second on startup
//	splash, stop := buildSplash(t.app)
//	t.app.SetRoot(splash, true)
//	time.AfterFunc(1*time.Second, func() {
//		stop()
//		t.app.QueueUpdateDraw(func() {
//			t.app.SetRoot(root, true)
//		})
//	})
//	t.app.EnableMouse(true)
//	if err := t.app.Run(); err != nil {
//		return err
//	}
//	return nil
//}
