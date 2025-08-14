package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

type Server struct {
	Alias    string
	Host     string
	User     string
	Port     int
	Key      string
	Tags     []string
	Status   string // "online", "warn", "offline"
	LastSeen time.Time
}

func mockServers() []Server {
	return []Server{
		{Alias: "web-01", Host: "192.168.1.10", User: "root", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "web"}, Status: "online", LastSeen: time.Now().Add(-2 * time.Hour)},
		{Alias: "web-02", Host: "192.168.1.11", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "web"}, Status: "warn", LastSeen: time.Now().Add(-30 * time.Minute)},
		{Alias: "db-01", Host: "192.168.1.20", User: "postgres", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "db"}, Status: "offline", LastSeen: time.Now().Add(-26 * time.Hour)},
		{Alias: "api-01", Host: "192.168.1.30", User: "deploy", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"prod", "api"}, Status: "online", LastSeen: time.Now().Add(-10 * time.Minute)},
		{Alias: "cache-01", Host: "192.168.1.40", User: "redis", Port: 22, Key: "~/.ssh/id_rsa", Tags: []string{"prod", "cache"}, Status: "online", LastSeen: time.Now().Add(-1 * time.Hour)},
		{Alias: "dev-web", Host: "10.0.1.10", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "web"}, Status: "online", LastSeen: time.Now().Add(-5 * time.Minute)},
		{Alias: "dev-db", Host: "10.0.1.20", User: "dev", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"dev", "db"}, Status: "online", LastSeen: time.Now().Add(-15 * time.Minute)},
		{Alias: "staging", Host: "staging.example.com", User: "ubuntu", Port: 22, Key: "~/.ssh/id_ed25519", Tags: []string{"test"}, Status: "warn", LastSeen: time.Now().Add(-45 * time.Minute)},
	}
}

func statusIcon(s string) string {
	switch s {
	case "online":
		return "🟢"
	case "warn":
		return "🟡"
	case "offline":
		return "🔴"
	default:
		return "⚪"
	}
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return "-"
	}
	return strings.Join(tags, ",")
}

func formatServerLine(s Server) (primary, secondary string) {
	icon := statusIcon(s.Status)
	// Choose a color per status for the alias and a subtle gray for host/time
	statusColor := "white"
	switch s.Status {
	case "online":
		statusColor = "green"
	case "warn":
		statusColor = "yellow"
	case "offline":
		statusColor = "red"
	}
	primary = fmt.Sprintf("%s [%s::b]%-12s[-] [#AAAAAA]%-18s[-] [#888888]Last:%s[-]", icon, statusColor, s.Alias, s.Host, humanizeDuration(time.Since(s.LastSeen)))
	secondary = ""
	return
}

func humanizeDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh%dm ago", h, m)
	}
	return fmt.Sprintf("%dm ago", m)
}

const AppName = "lazyssh"
const AppVersion = "v0.0.1"
const RepoURL = "github.com/adembc/lazyssh"

func runTUI() error {
	app := tview.NewApplication()

	// Global theme (non-invasive)
	tview.Styles.PrimitiveBackgroundColor = tcell.Color232 // near-black
	tview.Styles.ContrastBackgroundColor = tcell.Color235  // dark gray
	tview.Styles.BorderColor = tcell.Color238
	tview.Styles.TitleColor = tcell.Color250
	tview.Styles.PrimaryTextColor = tcell.Color252
	tview.Styles.TertiaryTextColor = tcell.Color245
	tview.Styles.SecondaryTextColor = tcell.Color245
	tview.Styles.GraphicsColor = tcell.Color238

	servers := mockServers()
	// Sort by alias for deterministic order
	sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })

	// Header: enhanced two-row layout (bar + separator)
	version := AppVersion
	repoURL := RepoURL

	// Slightly lighter than body for a distinct bar
	headerBg := tcell.Color234 // dark gray tint

	// Left: App icon + stylized name
	headerLeft := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	headerLeft.SetBackgroundColor(headerBg)
	stylizedName := "🚀 [#FFFFFF::b]lazy[-][#55D7FF::b]ssh[-]"
	headerLeft.SetText(stylizedName)

	// Center: badges (version + type)
	headerCenter := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	headerCenter.SetBackgroundColor(headerBg)
	// Render version as a pill/badge and add a secondary badge
	headerCenter.SetText("[black:#2ECC71::b]  " + version + "  [-]  [black:#3B82F6::b]  TUI  [-]")

	// Right: repo URL + current date/time
	headerRight := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	headerRight.SetBackgroundColor(headerBg)
	currentTime := time.Now().Format("Mon, 02 Jan 2006 15:04")
	headerRight.SetText("[#55AAFF::u]🔗 " + repoURL + "[-]  [#AAAAAA]• " + currentTime + "[-]")

	// Row 1: the main header bar with three columns
	headerBar := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(headerLeft, 0, 1, false).
		AddItem(headerCenter, 0, 1, false).
		AddItem(headerRight, 0, 1, false)

	// Row 2: a thin separator line
	separator := tview.NewTextView().SetDynamicColors(true)
	separator.SetBackgroundColor(tcell.Color235)
	separator.SetText("[#444444]" + strings.Repeat("─", 200) + "[-]")

	// Combine into a two-row header
	header := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(headerBar, 1, 0, false).
		AddItem(separator, 1, 0, false)

	// Tab bar removed per request

	// Left panel: Search is hidden by default; a hint bar is shown until '/' is pressed
	search := tview.NewInputField().
		SetLabel(" 🔍 Search: ").
		SetFieldWidth(30)
	// When visible, give it a border and title for better UI
	search.SetBorder(true).SetTitle("Search")
	search.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
	search.SetFieldBackgroundColor(tcell.Color233).SetFieldTextColor(tcell.Color252)

	serverList := tview.NewList().ShowSecondaryText(false)
	serverList.SetBorder(true)
	serverList.SetTitle("Servers")
	serverList.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
	serverList.SetSelectedBackgroundColor(tcell.Color24).SetSelectedTextColor(tcell.Color255)
	serverList.SetHighlightFullLine(true)

	// Hint bar shown when search is hidden
	hintBar := tview.NewTextView().SetDynamicColors(true)
	hintBar.SetBackgroundColor(tcell.Color233)
	hintBar.SetText("[#BBBBBB]Press [::b]/[-:-:b] to search…  •  ↑↓ Navigate  •  Enter SSH  •  a Add  •  e Edit  •  d Delete  •  ? Help[-]")

	// Right panel: Details
	details := tview.NewTextView().SetDynamicColors(true).SetWrap(true)
	details.SetBorder(true).SetTitle("Details")
	details.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)

	// Status bar
	status := tview.NewTextView().SetDynamicColors(true)
	statusBg := tcell.Color235
	status.SetBackgroundColor(statusBg)
	status.SetTextAlign(tview.AlignCenter)
	status.SetText("[white]↑↓[-] Navigate  • [white]Enter[-] SSH  • [white]a[-] Add  • [white]e[-] Edit  • [white]d[-] Delete  • [white]/[-] Search  • [white]q[-] Quit  • [white]?[-] Help")

	// Define updateDetails before populate; declare var for closure use
	var updateDetails func(Server)
	// Root container placeholder (assigned later)
	var root *tview.Flex

	// Populate list with filter logic
	filtered := servers
	populate := func() {
		serverList.Clear()
		for i := range filtered {
			p, s2 := formatServerLine(filtered[i])
			idx := i
			serverList.AddItem(p, s2, 0, func() {
				if updateDetails != nil {
					updateDetails(filtered[idx])
				}
			})
		}
		if serverList.GetItemCount() > 0 {
			serverList.SetCurrentItem(0)
			if updateDetails != nil {
				updateDetails(filtered[0])
			}
		} else {
			details.SetText("No servers match the current filter.")
		}
	}

	// Will be defined later to refresh according to current search
	var reapplyFilter func()

	// Delete currently selected server from memory and refresh
	deleteSelected := func() {
		idx := serverList.GetCurrentItem()
		if idx < 0 || idx >= len(filtered) {
			return
		}
		sel := filtered[idx]
		// remove from servers slice
		newServers := make([]Server, 0, len(servers))
		for _, s := range servers {
			if !(s.Alias == sel.Alias && s.Host == sel.Host) {
				newServers = append(newServers, s)
			}
		}
		servers = newServers
		reapplyFilter()
	}

	// Now define updateDetails
	updateDetails = func(s Server) {
		text := fmt.Sprintf("[::b]%s[-]\n\nHost: [white]%s[-]\nUser: [white]%s[-]\nPort: [white]%d[-]\nKey:  [white]%s[-]\nTags: [white]%s[-]\nStatus: %s\nLast: %s\n\n[::b]Commands:[-]\n  Enter: SSH connect\n  a: Add new server\n  e: Edit entry\n  d: Delete entry",
			s.Alias, s.Host, s.User, s.Port, s.Key, joinTags(s.Tags), statusIcon(s.Status), s.LastSeen.Format("2006-01-02 15:04"))
		details.SetText(text)
	}

	// Helper to re-apply current filter and repopulate
	reapplyFilter = func() {
		q := strings.TrimSpace(strings.ToLower(search.GetText()))
		if q == "" {
			filtered = servers
			populate()
			return
		}
		var out []Server
		for _, s := range servers {
			hay := strings.ToLower(strings.Join([]string{
				s.Alias,
				s.Host,
				s.User,
				fmt.Sprint(s.Port),
				strings.Join(s.Tags, ","),
				s.Key,
			}, " "))
			if strings.Contains(hay, q) {
				out = append(out, s)
			}
		}
		filtered = out
		populate()
	}

	// Add new server form
	addServer := func() {
		aliasField := tview.NewInputField().SetLabel("Alias: ")
		hostField := tview.NewInputField().SetLabel("Host/IP: ")
		userField := tview.NewInputField().SetLabel("User: ").SetText("root")
		portField := tview.NewInputField().SetLabel("Port: ").SetText("22")
		keyField := tview.NewInputField().SetLabel("Key: ").SetText("~/.ssh/id_ed25519")
		tagsField := tview.NewInputField().SetLabel("Tags (comma): ")
		statusDD := tview.NewDropDown().SetLabel("Status: ")
		statusOptions := []string{"online", "warn", "offline"}
		statusDD.SetOptions(statusOptions, nil)
		statusDD.SetCurrentOption(0)

		form := tview.NewForm().
			AddFormItem(aliasField).
			AddFormItem(hostField).
			AddFormItem(userField).
			AddFormItem(portField).
			AddFormItem(keyField).
			AddFormItem(tagsField).
			AddFormItem(statusDD)
		form.SetBorder(true).SetTitle("Add Server").SetTitleAlign(tview.AlignLeft)
		form.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
		form.AddButton("Save", func() {
			alias := strings.TrimSpace(aliasField.GetText())
			host := strings.TrimSpace(hostField.GetText())
			if alias == "" || host == "" {
				return
			}
			user := strings.TrimSpace(userField.GetText())
			p := 22
			if v := strings.TrimSpace(portField.GetText()); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					p = n
				}
			}
			key := strings.TrimSpace(keyField.GetText())
			tags := []string{}
			if ts := strings.TrimSpace(tagsField.GetText()); ts != "" {
				for _, t := range strings.Split(ts, ",") {
					if s := strings.TrimSpace(t); s != "" {
						tags = append(tags, s)
					}
				}
			}
			_, status := statusDD.GetCurrentOption()
			newS := Server{Alias: alias, Host: host, User: user, Port: p, Key: key, Tags: tags, Status: status, LastSeen: time.Now()}
			servers = append(servers, newS)
			sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
			reapplyFilter()
			// select the new item if it is visible
			for i := range filtered {
				if filtered[i].Alias == newS.Alias && filtered[i].Host == newS.Host {
					serverList.SetCurrentItem(i)
					updateDetails(filtered[i])
					break
				}
			}
			app.SetRoot(root, true)
		})
		form.AddButton("Cancel", func() { app.SetRoot(root, true) })
		form.SetCancelFunc(func() { app.SetRoot(root, true) })
		app.SetRoot(form, true)
	}

	// Edit currently selected server
	editSelected := func() {
		idx := serverList.GetCurrentItem()
		if idx < 0 || idx >= len(filtered) {
			return
		}
		orig := filtered[idx]
		aliasField := tview.NewInputField().SetLabel("Alias: ").SetText(orig.Alias)
		hostField := tview.NewInputField().SetLabel("Host/IP: ").SetText(orig.Host)
		userField := tview.NewInputField().SetLabel("User: ").SetText(orig.User)
		portField := tview.NewInputField().SetLabel("Port: ").SetText(fmt.Sprint(orig.Port))
		keyField := tview.NewInputField().SetLabel("Key: ").SetText(orig.Key)
		tagsField := tview.NewInputField().SetLabel("Tags (comma): ").SetText(strings.Join(orig.Tags, ", "))
		statusDD := tview.NewDropDown().SetLabel("Status: ")
		statusOptions := []string{"online", "warn", "offline"}
		statusDD.SetOptions(statusOptions, nil)
		// set current option to existing status
		curIdx := 0
		for i, o := range statusOptions {
			if o == orig.Status {
				curIdx = i
				break
			}
		}
		statusDD.SetCurrentOption(curIdx)

		form := tview.NewForm().
			AddFormItem(aliasField).
			AddFormItem(hostField).
			AddFormItem(userField).
			AddFormItem(portField).
			AddFormItem(keyField).
			AddFormItem(tagsField).
			AddFormItem(statusDD)
		form.SetBorder(true).SetTitle("Edit Server").SetTitleAlign(tview.AlignLeft)
		form.SetBorderColor(tcell.Color238).SetTitleColor(tcell.Color250)
		form.AddButton("Save", func() {
			alias := strings.TrimSpace(aliasField.GetText())
			host := strings.TrimSpace(hostField.GetText())
			if alias == "" || host == "" {
				return
			}
			user := strings.TrimSpace(userField.GetText())
			p := orig.Port
			if v := strings.TrimSpace(portField.GetText()); v != "" {
				if n, err := strconv.Atoi(v); err == nil && n > 0 {
					p = n
				}
			}
			key := strings.TrimSpace(keyField.GetText())
			tags := []string{}
			if ts := strings.TrimSpace(tagsField.GetText()); ts != "" {
				for _, t := range strings.Split(ts, ",") {
					if s := strings.TrimSpace(t); s != "" {
						tags = append(tags, s)
					}
				}
			}
			_, status := statusDD.GetCurrentOption()
			updated := Server{Alias: alias, Host: host, User: user, Port: p, Key: key, Tags: tags, Status: status, LastSeen: orig.LastSeen}
			// replace matching item in servers by original key alias+host
			found := false
			for i := range servers {
				if servers[i].Alias == orig.Alias && servers[i].Host == orig.Host {
					servers[i] = updated
					found = true
					break
				}
			}
			if !found {
				servers = append(servers, updated)
			}
			sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
			reapplyFilter()
			// select the updated item if visible
			for i := range filtered {
				if filtered[i].Alias == updated.Alias && filtered[i].Host == updated.Host {
					serverList.SetCurrentItem(i)
					updateDetails(filtered[i])
					break
				}
			}
			app.SetRoot(root, true)
		})
		form.AddButton("Cancel", func() { app.SetRoot(root, true) })
		form.SetCancelFunc(func() { app.SetRoot(root, true) })
		app.SetRoot(form, true)
	}

	filter := func(q string) {
		q = strings.TrimSpace(strings.ToLower(q))
		if q == "" {
			filtered = servers
			populate()
			return
		}
		var out []Server
		for _, s := range servers {
			hay := strings.ToLower(strings.Join([]string{
				s.Alias,
				s.Host,
				s.User,
				fmt.Sprint(s.Port),
				strings.Join(s.Tags, ","),
				s.Key,
			}, " "))
			if strings.Contains(hay, q) {
				out = append(out, s)
			}
		}
		filtered = out
		populate()
	}

	// Declare togglers; define after left is created
	var showSearchBar func()
	var hideSearchBar func()

	search.SetChangedFunc(func(text string) { filter(text) })
	search.SetDoneFunc(func(key tcell.Key) {
		// Return focus to list on Escape or Enter and hide the search bar
		if key == tcell.KeyEsc || key == tcell.KeyEnter {
			if hideSearchBar != nil {
				hideSearchBar()
			} else {
				app.SetFocus(serverList)
			}
		}
	})

	// Initial populate
	populate()

	// Layout main content area
	var left *tview.Flex
	left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(hintBar, 1, 0, false).
		AddItem(serverList, 0, 1, true)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(details, 0, 1, false)

	content := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(left, 0, 3, true).
		AddItem(right, 0, 2, false)

	// Define togglers now that 'left' exists
	showSearchBar = func() {
		left.Clear()
		left.AddItem(search, 3, 0, true)
		left.AddItem(serverList, 0, 1, false)
		app.SetFocus(search)
	}
	hideSearchBar = func() {
		left.Clear()
		left.AddItem(hintBar, 1, 0, false)
		left.AddItem(serverList, 0, 1, true)
		app.SetFocus(serverList)
	}

	// Root layout: header, content, status
	root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(header, 2, 0, false).
		AddItem(content, 0, 1, true).
		AddItem(status, 1, 0, false)

	// Keybindings
	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If the search input has focus, do not trigger global shortcuts.
		if app.GetFocus() == search {
			return event
		}
		switch event.Rune() {
		case 'q':
			app.Stop()
			return nil
		case '/':
			if showSearchBar != nil {
				showSearchBar()
			} else {
				app.SetFocus(search)
			}
			return nil
		case 'a':
			addServer()
			return nil
		case 'e':
			editSelected()
			return nil
		case 'd':
			deleteSelected()
			return nil
		case '?':
			showHelpModal(app, root)
			return nil
		}
		// Handle Enter at root level as connect for current selection
		if event.Key() == tcell.KeyEnter {
			idx := serverList.GetCurrentItem()
			if idx >= 0 && idx < len(filtered) {
				showConnectModal(app, root, filtered[idx])
				return nil
			}
		}
		return event
	})

	// Also handle list-selected changes to update details
	serverList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		if index >= 0 && index < len(filtered) {
			updateDetails(filtered[index])
		}
	})

	if err := app.SetRoot(root, true).EnableMouse(true).Run(); err != nil {
		return err
	}
	return nil
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

func showConnectModal(app *tview.Application, root tview.Primitive, s Server) {
	msg := fmt.Sprintf("SSH to %s (%s@%s:%d)\n\nThis is a mock action.", s.Alias, s.User, s.Host, s.Port)
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(root, true)
		})
	app.SetRoot(modal, true)
}

// printServers prints the list of servers to stdout in a simple table-like format.
func printServers() {
	servers := mockServers()
	sort.Slice(servers, func(i, j int) bool { return servers[i].Alias < servers[j].Alias })
	fmt.Printf("Alias\tHost\tUser\tPort\tTags\tStatus\tLast\n")
	for _, s := range servers {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\t%s\t%s\n", s.Alias, s.Host, s.User, s.Port, strings.Join(s.Tags, ","), s.Status, humanizeDuration(time.Since(s.LastSeen)))
	}
}

func main() {
	var (
		versionFlag bool
		listFlag    bool
	)

	rootCmd := &cobra.Command{
		Use:   AppName,
		Short: "Lazy SSH server picker TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionFlag {
				fmt.Println(AppVersion)
				return nil
			}
			if listFlag {
				printServers()
				return nil
			}
			return runTUI()
		},
	}
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	rootCmd.PersistentFlags().BoolVarP(&listFlag, "list", "l", false, "Print list of servers and exit")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(AppVersion)
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List servers",
		Run: func(cmd *cobra.Command, args []string) {
			printServers()
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
