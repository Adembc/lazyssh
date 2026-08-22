package ui

import (
	"strings"
	"testing"

	"github.com/Adembc/lazyssh/internal/core/domain"
	"github.com/Adembc/lazyssh/internal/core/ports"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type recordingServerService struct {
	ports.ServerService
	servers    []domain.Server
	sshCalls   int
	sshAliases []string
}

func (service *recordingServerService) ListServers(string) ([]domain.Server, error) {
	return service.servers, nil
}

func (service *recordingServerService) SSH(alias string) error {
	service.sshCalls++
	service.sshAliases = append(service.sshAliases, alias)
	return nil
}

type connectionConfirmationHarness struct {
	app        *tview.Application
	screen     tcell.SimulationScreen
	view       *tui
	service    *recordingServerService
	server     domain.Server
	serverList *ServerList
}

func newConnectionConfirmationHarness(t *testing.T) *connectionConfirmationHarness {
	t.Helper()

	server := domain.Server{Alias: "example"}
	service := &recordingServerService{servers: []domain.Server{server}}
	app := tview.NewApplication()
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("initialize simulation screen: %v", err)
	}
	t.Cleanup(screen.Fini)
	app.SetScreen(screen)

	serverList := NewServerList()
	serverList.UpdateServers(service.servers)
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("MAIN SERVER LIST"), 1, 0, false).
		AddItem(serverList, 0, 1, true)
	view := &tui{
		app:           app,
		serverService: service,
		serverList:    serverList,
		searchBar:     NewSearchBar(),
		root:          root,
	}
	app.SetRoot(root, true)
	app.SetFocus(serverList)

	return &connectionConfirmationHarness{
		app:        app,
		screen:     screen,
		view:       view,
		service:    service,
		server:     server,
		serverList: serverList,
	}
}

func openConnectionConfirmationOverlay(harness *connectionConfirmationHarness) (*tview.Modal, *tview.Pages) {
	modal, pages := harness.view.newConnectionConfirmationOverlay(harness.server)
	harness.app.SetRoot(pages, true)
	harness.app.SetFocus(modal)
	return modal, pages
}

func sendOverlayKey(app *tview.Application, pages *tview.Pages, event *tcell.EventKey) {
	pages.InputHandler()(event, func(primitive tview.Primitive) {
		app.SetFocus(primitive)
	})
}

func TestNormalizeGlobalHotkey(t *testing.T) {
	tests := map[rune]rune{
		'e': 'e',
		'E': 'e',
		'у': 'e',
		'У': 'e',
		's': 's',
		'ы': 's',
		'S': 'S',
		'Ы': 'S',
		'ф': 'a',
		'й': 'q',
		'.': '/',
		'1': '1',
	}

	for input, expected := range tests {
		if actual := normalizeGlobalHotkey(input); actual != expected {
			t.Fatalf("normalizeGlobalHotkey(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestCommandKeyNormalizesLayoutAndCaps(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected rune
	}{
		{name: "latin lower", input: 'd', expected: 'd'},
		{name: "latin caps", input: 'D', expected: 'd'},
		{name: "alternate layout lower", input: 'в', expected: 'd'},
		{name: "alternate layout caps", input: 'В', expected: 'd'},
		{name: "save lower", input: 'ы', expected: 's'},
		{name: "save caps", input: 'Ы', expected: 'S'},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := tcell.NewEventKey(tcell.KeyRune, test.input, tcell.ModNone)
			if actual := commandKey(event); actual != test.expected {
				t.Fatalf("commandKey(%q) = %q, want %q", test.input, actual, test.expected)
			}
		})
	}
}

func TestConnectionConfirmationAction(t *testing.T) {
	tests := []struct {
		name  string
		event *tcell.EventKey
		want  connectionConfirmationAction
	}{
		{name: "enter connects", event: tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), want: connectionConfirm},
		{name: "escape cancels", event: tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone), want: connectionCancel},
		{name: "english e edits", event: tcell.NewEventKey(tcell.KeyRune, 'E', tcell.ModNone), want: connectionEdit},
		{name: "alternate-layout e edits", event: tcell.NewEventKey(tcell.KeyRune, 'у', tcell.ModNone), want: connectionEdit},
		{name: "other key does nothing", event: tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone), want: connectionNoAction},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := connectionActionForKey(test.event); got != test.want {
				t.Fatalf("connectionActionForKey() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFirstEnterDoesNotConnect(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	harness.view.handleGlobalKeys(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if harness.service.sshCalls != 0 {
		t.Fatalf("first Enter called SSH %d time(s), want 0", harness.service.sshCalls)
	}
	if harness.app.GetFocus() == harness.serverList {
		t.Fatal("first Enter kept focus on server list, want connection confirmation modal button")
	}

	harness.app.ForceDraw()
	screenText := simulationScreenText(harness.screen)
	if !strings.Contains(screenText, "Confirm Connection") {
		t.Fatal("first Enter did not render the connection confirmation modal")
	}
	if !strings.Contains(screenText, "MAIN SERVER LIST") {
		t.Fatal("connection confirmation replaced the main UI instead of overlaying it")
	}
}

func TestConnectionConfirmationSecondEnterConnectsOnce(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	_, pages := openConnectionConfirmationOverlay(harness)
	harness.service.servers = append(harness.service.servers, domain.Server{Alias: "other"})
	harness.serverList.UpdateServers(harness.service.servers)
	harness.serverList.SetCurrentItem(1)

	sendOverlayKey(harness.app, pages, tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if harness.service.sshCalls != 1 {
		t.Fatalf("second Enter called SSH %d time(s), want 1", harness.service.sshCalls)
	}
	if got := harness.service.sshAliases[0]; got != harness.server.Alias {
		t.Fatalf("second Enter connected to %q, want displayed alias %q", got, harness.server.Alias)
	}
}

func TestConnectionConfirmationEditKeyOpensEditForm(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	_, pages := openConnectionConfirmationOverlay(harness)
	harness.service.servers = append(harness.service.servers, domain.Server{Alias: "other"})
	harness.serverList.UpdateServers(harness.service.servers)
	harness.serverList.SetCurrentItem(1)

	sendOverlayKey(harness.app, pages, tcell.NewEventKey(tcell.KeyRune, 'У', tcell.ModNone))
	harness.app.ForceDraw()

	if harness.service.sshCalls != 0 {
		t.Fatalf("edit key called SSH %d time(s), want 0", harness.service.sshCalls)
	}
	screenText := simulationScreenText(harness.screen)
	if !strings.Contains(screenText, "Edit Server") {
		t.Fatal("Caps/alternate-layout E equivalent did not open the edit form")
	}
	if !strings.Contains(screenText, harness.server.Alias) {
		t.Fatalf("edit form did not retain displayed alias %q after selection changed", harness.server.Alias)
	}
}

func TestConnectionConfirmationEscapeReturnsToServerList(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	_, pages := openConnectionConfirmationOverlay(harness)

	sendOverlayKey(harness.app, pages, tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	harness.app.ForceDraw()

	if harness.service.sshCalls != 0 {
		t.Fatalf("Escape called SSH %d time(s), want 0", harness.service.sshCalls)
	}
	if harness.app.GetFocus() != harness.serverList {
		t.Fatal("Escape did not restore focus to the server list")
	}
	screenText := simulationScreenText(harness.screen)
	if !strings.Contains(screenText, "MAIN SERVER LIST") || strings.Contains(screenText, "Confirm Connection") {
		t.Fatal("Escape did not restore the main server-list view")
	}
}

func TestConnectionConfirmationBlocksClicksBehindModal(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	modal, pages := openConnectionConfirmationOverlay(harness)
	harness.app.ForceDraw()

	if modal.InRect(0, 0) {
		t.Fatal("test coordinate unexpectedly falls inside the centered modal")
	}
	consumed, _ := pages.MouseHandler()(
		tview.MouseLeftDown,
		tcell.NewEventMouse(0, 0, tcell.Button1, tcell.ModNone),
		func(primitive tview.Primitive) { harness.app.SetFocus(primitive) },
	)
	if !consumed {
		t.Fatal("click outside confirmation modal was allowed through to the main UI")
	}
}

func TestConnectionConfirmationMessage(t *testing.T) {
	const want = "You are about to connect to \"a\\\"b\".\n\nTo edit this item instead, press E."
	if got := connectionConfirmationMessage(`a"b`); got != want {
		t.Fatalf("connectionConfirmationMessage() = %q, want %q", got, want)
	}
}

func TestConnectionConfirmationEscapesAliasTags(t *testing.T) {
	harness := newConnectionConfirmationHarness(t)
	harness.server.Alias = "[red]prod"
	harness.view.showConnectionConfirmModal(harness.server)
	harness.app.ForceDraw()

	if !strings.Contains(simulationScreenText(harness.screen), harness.server.Alias) {
		t.Fatal("connection confirmation interpreted alias text as a tview color tag")
	}
}

func simulationScreenText(screen tcell.SimulationScreen) string {
	width, height := screen.Size()
	var text strings.Builder
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			mainc, _, _, _ := screen.GetContent(x, y)
			text.WriteRune(mainc)
		}
	}
	return text.String()
}
