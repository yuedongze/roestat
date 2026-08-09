package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yuedongze/roestat/internal/roest"
)

type viewState int

const (
	viewHistory viewState = iota
	viewDetail
	viewPicker
	viewLive
)

// App is the root Bubble Tea model. It owns the sub-views and routes messages
// and navigation between them.
type App struct {
	client *roest.Client
	state  viewState
	w, h   int

	history historyModel
	detail  detailModel
	picker  pickerModel
	live    liveModel
}

// NewApp constructs the root model.
func NewApp(c *roest.Client) App {
	return App{
		client:  c,
		state:   viewHistory,
		history: newHistory(c),
		detail:  newDetail(c),
		picker:  newPicker(),
		live:    newLive(),
	}
}

func (a App) Init() tea.Cmd {
	// Load the first page of history and start the 1s clock for the live view.
	return tea.Batch(loadLogsPage(a.client, 1), tick())
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		a.layout()
		return a, nil

	case tea.KeyMsg:
		return a.handleKey(msg)

	case machinesMsg:
		if msg.err != nil {
			a.picker.err = msg.err
		} else {
			a.picker.setMachines(msg.machines)
		}
		return a, nil

	case logsPageMsg:
		if msg.err != nil {
			a.history.err = msg.err
			a.history.loading = false
			a.history.loadingMore = false
		} else {
			a.history.addLogs(msg.logs, msg.page, msg.hasNext)
		}
		return a, nil

	case datapointsMsg:
		if msg.logID == a.detail.log.ID {
			if msg.err != nil {
				a.detail.err = msg.err
			} else {
				a.detail.points = msg.points
			}
			a.detail.loading = false
		}
		return a, nil

	case liveConnectedMsg:
		a.live.connecting = false
		if msg.err != nil {
			a.live.err = msg.err
			return a, nil
		}
		a.live.live = msg.client
		return a, waitForLive(msg.client)

	case liveDataMsg:
		p := roest.LivePayload(msg)
		a.live.ingest(p)
		var cmds []tea.Cmd
		if a.live.live != nil {
			cmds = append(cmds, waitForLive(a.live.live))
		}
		// The first live sample confirms a roast is running; seed the chart with
		// the datapoints collected before we connected.
		if !a.live.backfillRequested {
			a.live.backfillRequested = true
			cmds = append(cmds, loadLiveBackfill(a.client, a.live.machine, p.ChargeTimestamp, a.live.gen))
		}
		return a, tea.Batch(cmds...)

	case backfillMsg:
		// Ignore results for a roast the user has since navigated away from.
		if a.state != viewLive || msg.gen != a.live.gen || msg.machine.ID != a.live.machine.ID {
			return a, nil
		}
		a.live.applyBackfill(msg)
		return a, nil

	case tickMsg:
		// Keep the clock ticking; the view reads elapsed time from live data.
		return a, tick()
	}

	// Forward anything else to the active view.
	return a, a.routeToActive(msg)
}

// handleKey processes global keys, then delegates to the active view.
func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		a.live.stop()
		return a, tea.Quit
	}

	switch a.state {
	case viewHistory:
		switch msg.String() {
		case "q":
			return a, tea.Quit
		case "l":
			a.state = viewPicker
			a.picker.loading = true
			a.layout()
			return a, loadMachines(a.client)
		case "enter":
			if l, ok := a.history.selectedLog(); ok {
				a.detail.open(l)
				a.state = viewDetail
				a.layout()
				return a, loadDatapoints(a.client, l.ID)
			}
		}
		return a, a.history.update(msg)

	case viewDetail:
		switch msg.String() {
		case "esc", "backspace":
			a.state = viewHistory
			a.layout()
			return a, nil
		case "q":
			return a, tea.Quit
		}
		return a, nil

	case viewPicker:
		// While the filter input is active, let the list consume all keys.
		if a.picker.filtering() {
			return a, a.picker.update(msg)
		}
		switch msg.String() {
		case "esc":
			a.state = viewHistory
			a.layout()
			return a, nil
		case "q":
			return a, tea.Quit
		case "enter":
			if m, ok := a.picker.selected(); ok {
				a.live.begin(m)
				a.state = viewLive
				a.layout()
				return a, connectLive(m)
			}
		}
		return a, a.picker.update(msg)

	case viewLive:
		switch msg.String() {
		case "esc", "backspace":
			a.live.stop()
			a.state = viewPicker
			a.layout()
			return a, nil
		case "q":
			a.live.stop()
			return a, tea.Quit
		}
		return a, nil
	}
	return a, nil
}

// routeToActive forwards non-key messages to the focused view when relevant.
func (a *App) routeToActive(msg tea.Msg) tea.Cmd {
	switch a.state {
	case viewHistory:
		return a.history.update(msg)
	case viewPicker:
		return a.picker.update(msg)
	}
	return nil
}

// layout distributes the terminal size to the sub-views, reserving rows for the
// title bar and help footer.
func (a *App) layout() {
	contentH := max(a.h-2, 3)
	a.history.setSize(a.w, contentH-1)
	a.detail.setSize(a.w, contentH)
	a.picker.setSize(a.w, contentH)
	a.live.setSize(a.w, contentH)
}

func (a App) View() string {
	if a.w == 0 {
		return "Starting roestat…"
	}

	var body, help string
	switch a.state {
	case viewHistory:
		body = a.history.view()
		help = "↑/↓ move · enter details · l live view · q quit"
	case viewDetail:
		body = a.detail.view()
		help = "esc back · q quit"
	case viewPicker:
		body = a.picker.view()
		help = "↑/↓ move · / filter · enter watch · esc back"
	case viewLive:
		body = a.live.view()
		help = "esc back to picker · q quit"
	}

	title := titleStyle.Render("roestat") + lipgloss.NewStyle().Foreground(colorDim).Render("  ROEST roaster monitor")
	footer := helpStyle.Render(help)

	return strings.Join([]string{title, body, footer}, "\n")
}
