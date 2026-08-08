package ui

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/charmbracelet/lipgloss"

	"github.com/yuedongze/roestat/internal/roest"
)

// liveModel is the real-time dashboard for one machine's roast.
type liveModel struct {
	machine    roest.Machine
	live       *roest.LiveClient
	connecting bool
	err        error

	last     roest.LivePayload
	haveData bool

	samples []roest.Sample // (msec, bt) history for RoR
	btPts   []canvas.Float64Point
	etPts   []canvas.Float64Point
	tgPts   []canvas.Float64Point
	ror     float64
	haveRoR bool
	maxBT   float64

	w, h int
}

func newLive() liveModel { return liveModel{} }

func (m *liveModel) setSize(w, h int) { m.w, m.h = w, h }

// begin resets state for a freshly selected machine.
func (m *liveModel) begin(machine roest.Machine) {
	m.machine = machine
	m.live = nil
	m.connecting = true
	m.err = nil
	m.last = roest.LivePayload{}
	m.haveData = false
	m.samples = nil
	m.btPts, m.etPts, m.tgPts = nil, nil, nil
	m.ror, m.haveRoR, m.maxBT = 0, false, 0
}

// stop closes the MQTT subscription.
func (m *liveModel) stop() {
	if m.live != nil {
		m.live.Close()
		m.live = nil
	}
}

// ingest records a new live payload and recomputes derived series.
func (m *liveModel) ingest(p roest.LivePayload) {
	m.last = p
	m.haveData = true
	d := p.Data
	x := float64(d.Msec) / 1000
	m.btPts = append(m.btPts, canvas.Float64Point{X: x, Y: d.BT})
	m.etPts = append(m.etPts, canvas.Float64Point{X: x, Y: d.ET})
	m.tgPts = append(m.tgPts, canvas.Float64Point{X: x, Y: d.Target})
	m.samples = append(m.samples, roest.Sample{Msec: d.Msec, BT: d.BT})
	if d.BT > m.maxBT {
		m.maxBT = d.BT
	}
	m.ror, m.haveRoR = roest.RoR(m.samples, roest.DefaultRoRWindowSec)
}

// phase returns the current roast phase from the latest event.
func (m liveModel) phase() string {
	phase := "—"
	best := -1
	for _, e := range m.last.Events {
		if e.Msec >= best {
			best = e.Msec
			phase = eventName(e.Type)
		}
	}
	return phase
}

func eventName(t int) string {
	switch t {
	case 0:
		return "Charge"
	case 1:
		return "Drop"
	case 4:
		return "Dry end"
	case 5:
		return "First crack"
	default:
		return fmt.Sprintf("Event %d", t)
	}
}

func (m liveModel) view() string {
	head := titleStyle.Render("● LIVE · " + m.machine.Name)

	if m.err != nil {
		return head + "\n\n" + errorStyle.Render(m.err.Error())
	}
	if m.connecting {
		return head + "\n\n" + spinnerHintStyle.Render("Connecting to "+roest.MQTTBroker+"…")
	}
	if !m.haveData {
		return head + "\n\n" + spinnerHintStyle.Render("Connected. Waiting for the roaster to publish (is a roast running?)…")
	}

	d := m.last.Data
	elapsed := d.Msec / 1000
	rorStr := "—"
	if m.haveRoR {
		rorStr = fmt.Sprintf("%+.1f", m.ror)
	}

	gauges := statGrid([][2]string{
		{"Time", fmt.Sprintf("%d:%02d", elapsed/60, elapsed%60)},
		{"Phase", m.phase()},
		{"BT", fmt.Sprintf("%.1f°C", d.BT)},
		{"ET", fmt.Sprintf("%.1f°C", d.ET)},
		{"Target", fmt.Sprintf("%.1f°C", d.Target)},
		{"RoR", rorStr + "°C/min"},
		{"Heat", fmt.Sprintf("%.0f%%", d.Heat)},
		{"Fan", fmt.Sprintf("%.0f%%", d.Fan)},
		{"Drum", fmt.Sprintf("%.0f rpm", d.RPM)},
		{"Max BT", fmt.Sprintf("%.0f°C", m.maxBT)},
	}, 5)

	legend := tempLegend()
	chartH := max(m.h-lipgloss.Height(gauges)-4, 6)
	chart := renderChart(m.w, chartH, []chartSeries{
		{name: "target", style: targetLineStyle, points: m.tgPts},
		{name: "et", style: etLineStyle, points: m.etPts},
		{name: "bt", style: btLineStyle, points: m.btPts},
	})

	return strings.Join([]string{head, gauges, legend, chart}, "\n")
}
