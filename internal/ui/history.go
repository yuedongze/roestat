package ui

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yuedongze/roestat/internal/roest"
)

// historyModel is the browsable table of past roasts, with lazy pagination.
type historyModel struct {
	client      *roest.Client
	table       table.Model
	logs        []roest.Log
	page        int
	hasNext     bool
	loading     bool // initial page load
	loadingMore bool // fetching a subsequent page
	err         error
	w, h        int
}

func newHistory(c *roest.Client) historyModel {
	cols := []table.Column{
		{Title: "Batch", Width: 7},
		{Title: "Bean", Width: 28},
		{Title: "Date", Width: 16},
		{Title: "Duration", Width: 9},
		{Title: "Loss", Width: 6},
		{Title: "FC °C", Width: 6},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
	)

	// High-contrast selection: bold dark text on the orange accent, so the
	// current row is obvious at a glance.
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(colorAccent).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorAxis).
		BorderBottom(true)
	s.Selected = s.Selected.
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(colorAccent)
	t.SetStyles(s)

	return historyModel{client: c, table: t, loading: true}
}

func (m *historyModel) setSize(w, h int) {
	m.w, m.h = w, h
	m.table.SetWidth(w)
	m.table.SetHeight(h)
}

// addLogs appends a fetched page and rebuilds the table rows.
func (m *historyModel) addLogs(logs []roest.Log, page int, hasNext bool) {
	m.logs = append(m.logs, logs...)
	m.page = page
	m.hasNext = hasNext
	m.loading = false
	m.loadingMore = false

	rows := make([]table.Row, len(m.logs))
	for i, l := range m.logs {
		date := "—"
		if t, ok := l.StartTime(); ok {
			date = t.Local().Format("2006-01-02 15:04")
		}
		dur := "—"
		if d, ok := l.Duration(); ok {
			dur = fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
		}
		loss := "—"
		if pct, ok := l.WeightLossPct(); ok {
			loss = fmt.Sprintf("%.1f%%", pct)
		}
		fc := "—"
		if l.FCTemp != nil {
			fc = fmt.Sprintf("%.0f", *l.FCTemp)
		}
		rows[i] = table.Row{
			strconv.Itoa(l.BatchNo),
			truncateStr(l.Bean(), 28),
			date, dur, loss, fc,
		}
	}
	m.table.SetRows(rows)
}

// update handles table navigation and triggers lazy loading of the next page
// when the cursor approaches the end of what's loaded.
func (m *historyModel) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	if !m.loadingMore && m.hasNext && m.table.Cursor() >= len(m.logs)-5 {
		m.loadingMore = true
		return tea.Batch(cmd, loadLogsPage(m.client, m.page+1))
	}
	return cmd
}

func (m historyModel) selectedLog() (roest.Log, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.logs) {
		return roest.Log{}, false
	}
	return m.logs[i], true
}

func (m historyModel) view() string {
	if m.err != nil {
		return errorStyle.Render("Failed to load roasts: " + m.err.Error())
	}
	if m.loading {
		return spinnerHintStyle.Render("Loading roast history…")
	}
	footer := fmt.Sprintf("%d roasts loaded", len(m.logs))
	if m.loadingMore {
		footer += " · loading more…"
	} else if m.hasNext {
		footer += " · scroll for more"
	}
	return m.table.View() + "\n" + spinnerHintStyle.Render(footer)
}

func truncateStr(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
