package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yuedongze/roestat/internal/roest"
)

// Async result messages.
type (
	machinesMsg struct {
		machines []roest.Machine
		err      error
	}
	logsPageMsg struct {
		logs    []roest.Log
		page    int
		hasNext bool
		err     error
	}
	datapointsMsg struct {
		logID  int
		points []roest.Datapoint
		err    error
	}
	liveConnectedMsg struct {
		machine roest.Machine
		client  *roest.LiveClient
		err     error
	}
	liveDataMsg roest.LivePayload
	tickMsg     time.Time
)

// loadMachines fetches the machine list for the picker.
func loadMachines(c *roest.Client) tea.Cmd {
	return func() tea.Msg {
		m, err := c.GetMachines()
		return machinesMsg{machines: m, err: err}
	}
}

// loadLogsPage fetches one page of roast history.
func loadLogsPage(c *roest.Client, page int) tea.Cmd {
	return func() tea.Msg {
		logs, hasNext, err := c.GetLogsPage(page)
		return logsPageMsg{logs: logs, page: page, hasNext: hasNext, err: err}
	}
}

// loadDatapoints fetches the full curve for a roast log.
func loadDatapoints(c *roest.Client, logID int) tea.Cmd {
	return func() tea.Msg {
		pts, err := c.GetDatapoints(logID)
		return datapointsMsg{logID: logID, points: pts, err: err}
	}
}

// connectLive opens the MQTT subscription for a machine.
func connectLive(m roest.Machine) tea.Cmd {
	return func() tea.Msg {
		lc, err := roest.ConnectLive(m)
		return liveConnectedMsg{machine: m, client: lc, err: err}
	}
}

// waitForLive blocks on the next MQTT payload and delivers it to the program.
// The live model re-issues this after each message to keep the stream flowing.
func waitForLive(lc *roest.LiveClient) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-lc.Messages
		if !ok {
			return nil
		}
		return liveDataMsg(p)
	}
}

// tick drives the once-per-second elapsed-time clock.
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}
