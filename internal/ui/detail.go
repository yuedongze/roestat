package ui

import (
	"fmt"
	"strings"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/charmbracelet/lipgloss"

	"github.com/yuedongze/roestat/internal/roest"
)

// detailModel shows one historical roast: summary stats plus its curves.
type detailModel struct {
	client  *roest.Client
	log     roest.Log
	points  []roest.Datapoint
	loading bool
	err     error
	w, h    int
}

func newDetail(c *roest.Client) detailModel {
	return detailModel{client: c}
}

func (m *detailModel) setSize(w, h int) { m.w, m.h = w, h }

// open resets the model for a newly selected log and returns the fetch command.
func (m *detailModel) open(l roest.Log) {
	m.log = l
	m.points = nil
	m.err = nil
	m.loading = true
}

func (m detailModel) view() string {
	title := fmt.Sprintf("Batch #%d · %s", m.log.BatchNo, m.log.Bean())
	head := titleStyle.Render(title)

	if m.err != nil {
		return head + "\n\n" + errorStyle.Render("Failed to load datapoints: "+m.err.Error())
	}
	if m.loading {
		return head + "\n\n" + spinnerHintStyle.Render("Loading roast curve…")
	}

	stats := m.statsBlock()

	// Split remaining height between the temperature chart (upper, larger) and
	// the RoR chart (lower). Reserve rows for the title, stats, and legend.
	chartH := max(m.h-lipgloss.Height(stats)-4, 6)
	tempH := chartH * 2 / 3
	rorH := chartH - tempH

	tempChart := renderChart(m.w, tempH, m.tempSeries())
	// Zoom the RoR chart to its positive range; the negative tail is less useful.
	rorChart := renderChart(m.w, rorH, m.rorSeries(), withYFloor(0))

	legend := tempLegend() + "   " + legendRoRStyle.Render("■ RoR (°C/min)")

	return strings.Join([]string{
		head,
		stats,
		legend,
		tempChart,
		rorChart,
	}, "\n")
}

func (m detailModel) statsBlock() string {
	dur := "—"
	if d, ok := m.log.Duration(); ok {
		dur = fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	}
	loss := "—"
	if pct, ok := m.log.WeightLossPct(); ok {
		loss = fmt.Sprintf("%.1f%%", pct)
	}
	maxBT := 0.0
	for _, p := range m.points {
		if p.BT != nil && *p.BT > maxBT {
			maxBT = *p.BT
		}
	}
	pairs := [][2]string{
		{"Profile", m.log.Profile()},
		{"Duration", dur},
		{"Weight loss", loss},
		{"Max BT", fmt.Sprintf("%.0f°C", maxBT)},
		{"FC", optTemp(m.log.FCTemp)},
		{"End", optTemp(m.log.EndTemp)},
	}
	return statLine(pairs)
}

func (m detailModel) tempSeries() []chartSeries {
	var bt, et, tg []canvas.Float64Point
	for _, p := range m.points {
		x := float64(p.Msec) / 1000
		if p.BT != nil {
			bt = append(bt, canvas.Float64Point{X: x, Y: *p.BT})
		}
		if p.ET != nil {
			et = append(et, canvas.Float64Point{X: x, Y: *p.ET})
		}
		if p.Target != nil {
			tg = append(tg, canvas.Float64Point{X: x, Y: *p.Target})
		}
	}
	return []chartSeries{
		{name: "target", style: targetLineStyle, points: tg},
		{name: "et", style: etLineStyle, points: et},
		{name: "bt", style: btLineStyle, points: bt},
	}
}

func (m detailModel) rorSeries() []chartSeries {
	var ror []canvas.Float64Point
	var samples []roest.Sample
	for _, p := range m.points {
		// Prefer the API's RoR; fall back to computing it from bean-temp deltas
		// (the same 30s-window method the live view uses), since REST datapoints
		// often come back with no RoR.
		if v, ok := p.RoR(); ok {
			ror = append(ror, canvas.Float64Point{X: float64(p.Msec) / 1000, Y: v})
			continue
		}
		if p.BT == nil {
			continue
		}
		samples = append(samples, roest.Sample{Msec: p.Msec, BT: *p.BT})
		if v, ok := roest.RoR(samples, roest.DefaultRoRWindowSec); ok {
			ror = append(ror, canvas.Float64Point{X: float64(p.Msec) / 1000, Y: v})
		}
	}
	return []chartSeries{{name: "ror", style: rorLineStyle, points: ror}}
}

func optTemp(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f°C", *v)
}
