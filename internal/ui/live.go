package ui

import (
	"fmt"
	"sort"
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

	// Backfill of the roast so far, fetched over REST once the first live sample
	// confirms a roast is running. gen guards against a slow fetch landing after
	// the user navigated away; backfillMax is the newest backfilled Msec (-1 when
	// none has been applied), used to dedup overlapping live samples.
	gen               int
	backfillRequested bool
	backfillMax       int

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
	m.gen++
	m.backfillRequested = false
	m.backfillMax = -1
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
	// Skip samples that fall inside the backfilled range so live data can't
	// duplicate points the REST seed already covered.
	if m.backfillMax >= 0 && p.Data.Msec <= m.backfillMax {
		return
	}
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

// applyBackfill seeds the chart with the datapoints collected before the live
// view connected, merging them ahead of the live samples already plotted. It is
// order-independent: the backfill owns the range it covers and the live stream
// only keeps points newer than the newest backfilled Msec, so it works whether
// this lands before or after the first live samples. A missing/empty backfill is
// silent — the view simply stays live-only.
func (m *liveModel) applyBackfill(msg backfillMsg) {
	if msg.err != nil || !msg.found || len(msg.points) == 0 {
		return
	}

	pts := append([]roest.Datapoint(nil), msg.points...)
	sort.Slice(pts, func(i, j int) bool { return pts[i].Msec < pts[j].Msec })

	var bt, et, tg []canvas.Float64Point
	var samples []roest.Sample
	backfillMax := 0
	for _, p := range pts {
		x := float64(p.Msec) / 1000
		if p.BT != nil {
			bt = append(bt, canvas.Float64Point{X: x, Y: *p.BT})
			samples = append(samples, roest.Sample{Msec: p.Msec, BT: *p.BT})
		}
		if p.ET != nil {
			et = append(et, canvas.Float64Point{X: x, Y: *p.ET})
		}
		if p.Target != nil {
			tg = append(tg, canvas.Float64Point{X: x, Y: *p.Target})
		}
		if p.Msec > backfillMax {
			backfillMax = p.Msec
		}
	}
	m.backfillMax = backfillMax

	// Drop any live samples that overlap the backfilled range, keeping only the
	// tail newer than backfillMax. The four live slices are appended in lockstep,
	// so a single cut index (found via samples) applies to all of them.
	cut := len(m.samples)
	for i, s := range m.samples {
		if s.Msec > backfillMax {
			cut = i
			break
		}
	}
	m.btPts = append(bt, m.btPts[cut:]...)
	m.etPts = append(et, m.etPts[cut:]...)
	m.tgPts = append(tg, m.tgPts[cut:]...)
	m.samples = append(samples, m.samples[cut:]...)

	m.maxBT = 0
	for _, p := range m.btPts {
		if p.Y > m.maxBT {
			m.maxBT = p.Y
		}
	}
	m.ror, m.haveRoR = roest.RoR(m.samples, roest.DefaultRoRWindowSec)
	m.haveData = true
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
		{"Last event", m.phase()},
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
	phaseSection := m.phaseBlock() // "" until the roast has an elapsed time

	reserved := lipgloss.Height(gauges) + 4
	if phaseSection != "" {
		reserved += lipgloss.Height(phaseSection)
	}
	chartH := max(m.h-reserved, 6)
	chart := renderChart(m.w, chartH, []chartSeries{
		{name: "target", style: targetLineStyle, points: m.tgPts},
		{name: "et", style: etLineStyle, points: m.etPts},
		{name: "bt", style: btLineStyle, points: m.btPts},
	})

	parts := []string{head, gauges, legend, chart}
	if phaseSection != "" {
		parts = append(parts, phaseSection)
	}
	return strings.Join(parts, "\n")
}

// phaseBlock renders the growing Drying/Maillard/Development bar with its legend,
// or "" when there's no elapsed roast time yet.
func (m liveModel) phaseBlock() string {
	phases, ok := roest.LivePhases(m.last.Events, m.last.Data.Msec)
	if !ok {
		return ""
	}
	return phaseLegend() + "\n" + phaseBar(m.w, phases)
}
