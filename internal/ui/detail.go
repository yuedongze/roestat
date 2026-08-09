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
	phaseSection := m.phaseBlock() // "" when phase boundaries aren't available

	// The top chart overlays temperature + RoR (dual axis) and gets the larger
	// share; the bottom chart shows fan/power. Reserve rows for the title, stats,
	// two legends, and the phase bar when shown.
	reserved := lipgloss.Height(stats) + 5
	if phaseSection != "" {
		reserved += lipgloss.Height(phaseSection)
	}
	chartH := max(m.h-reserved, 8)
	topH := chartH * 2 / 3
	botH := chartH - topH

	topLegend := tempLegend() + "   " + legendRoRStyle.Render("■ RoR ("+rorSuffix()+")")
	botLegend := legendFanStyle.Render("■ Fan %") + "   " + legendPowerStyle.Render("■ Power %")

	topChart := renderRoRDual(m.w, topH, m.tempSeries(), m.rorData())
	botChart := renderChart(m.w, botH, m.powerSeries(), withYFloor(0))

	parts := []string{head, stats, topLegend, topChart, botLegend, botChart}
	if phaseSection != "" {
		parts = append(parts, phaseSection)
	}
	return strings.Join(parts, "\n")
}

// phaseBlock renders the Drying/Maillard/Development bar with its legend, or ""
// when the roast's phase boundaries aren't available.
func (m detailModel) phaseBlock() string {
	phases, ok := m.log.Phases()
	if !ok {
		return ""
	}
	return phaseLegend() + "\n" + phaseBar(m.w, phases)
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
		{"Max BT", fmt.Sprintf("%.0f%s", convTemp(maxBT), tempSuffix())},
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
		{name: "target", style: targetLineStyle, points: tg, kind: valueTemp},
		{name: "et", style: etLineStyle, points: et, kind: valueTemp},
		{name: "bt", style: btLineStyle, points: bt, kind: valueTemp},
	}
}

// rorData returns the RoR curve (°C/min) for overlaying on the temperature
// chart. It prefers the API's RoR and falls back to computing it from bean-temp
// deltas (the same 30s-window method the live view uses), since REST datapoints
// often come back with no RoR.
func (m detailModel) rorData() []canvas.Float64Point {
	var ror []canvas.Float64Point
	var samples []roest.Sample
	for _, p := range m.points {
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
	return ror
}

// powerSeries returns the fan-speed and heater-power curves (both percentages).
func (m detailModel) powerSeries() []chartSeries {
	var fan, power []canvas.Float64Point
	for _, p := range m.points {
		x := float64(p.Msec) / 1000
		if p.Fan != nil {
			fan = append(fan, canvas.Float64Point{X: x, Y: *p.Fan})
		}
		if p.Heat != nil {
			power = append(power, canvas.Float64Point{X: x, Y: *p.Heat})
		}
	}
	return []chartSeries{
		{name: "fan", style: fanLineStyle, points: fan},
		{name: "power", style: powerLineStyle, points: power},
	}
}

func optTemp(v *float64) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%.0f%s", convTemp(*v), tempSuffix())
}
