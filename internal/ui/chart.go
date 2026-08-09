package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/linechart"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
)

// chartBase anchors the time axis: point X (elapsed seconds) is mapped to
// chartBase + X seconds, so the axis float value equals elapsed seconds.
var chartBase = time.Unix(0, 0).UTC()

// valueKind tells renderChart how to unit-convert a series' Y values: raw (no
// conversion), an absolute temperature, or a temperature rate (RoR).
type valueKind int

const (
	valueRaw valueKind = iota
	valueTemp
	valueRate
)

// chartSeries is one named line to draw (X = elapsed seconds, Y = value in °C).
type chartSeries struct {
	name   string
	style  lipgloss.Style
	points []canvas.Float64Point
	kind   valueKind
}

// convValue applies the current unit to a series value according to its kind.
func (s chartSeries) convValue(y float64) float64 {
	switch s.kind {
	case valueTemp:
		return convTemp(y)
	case valueRate:
		return convDelta(y)
	default:
		return y
	}
}

// chartOpts tunes an individual chart; zero value auto-fits the axes.
type chartOpts struct {
	yFloor *float64 // clamp the visible Y minimum to this value when set
}

// chartOption configures renderChart.
type chartOption func(*chartOpts)

// withYFloor clamps the chart's visible Y minimum, hiding anything below v (e.g.
// floor at 0 to zoom in on the positive part of a curve).
func withYFloor(v float64) chartOption {
	return func(o *chartOpts) { o.yFloor = &v }
}

// elapsedLabel formats an X-axis value (elapsed seconds) as m:ss.
func elapsedLabel(_ int, v float64) string {
	s := max(int(v), 0)
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// renderChart builds a fresh time-series line chart sized w×h and draws every
// series as a smooth braille curve, auto-fitting the axes to the data. It is
// stateless: cheap enough to call once per redraw for the point counts a roast
// produces (~1/sec, <1000 points).
func renderChart(w, h int, series []chartSeries, opts ...chartOption) string {
	if w < 8 || h < 4 {
		return ""
	}
	var cfg chartOpts
	for _, o := range opts {
		o(&cfg)
	}
	minY, maxY, ok := yBounds(series, cfg.yFloor)
	if !ok {
		return waitingBox(w, h)
	}
	return chartCore(w, h, series, minY, maxY)
}

// yBounds computes the padded, unit-converted Y range across every series' data,
// optionally clamped to a floor. ok is false when there are no points.
func yBounds(series []chartSeries, floor *float64) (minY, maxY float64, ok bool) {
	seen := false
	for _, s := range series {
		for _, p := range s.points {
			y := s.convValue(p.Y)
			if !seen {
				minY, maxY, seen = y, y, true
				continue
			}
			minY = min(minY, y)
			maxY = max(maxY, y)
		}
	}
	if !seen {
		return 0, 0, false
	}
	pad := max((maxY-minY)*0.1, 1)
	minY -= pad
	maxY += pad
	// Clamp the visible floor (e.g. hide negative RoR) and keep a non-empty range.
	if floor != nil && minY < *floor {
		minY = *floor
		if maxY <= minY {
			maxY = minY + 1
		}
	}
	return minY, maxY, true
}

// chartCore renders the given series against a fixed Y range [minY,maxY]. The Y
// range is locked (the chart won't auto-expand to fit out-of-range points, so a
// clamped floor holds and overlaid series can't stretch the axis); the X range
// is auto-fit to the data.
func chartCore(w, h int, series []chartSeries, minY, maxY float64) string {
	minX, maxX, ok := xBounds(series)
	if !ok {
		return waitingBox(w, h)
	}
	minT := chartBase.Add(time.Duration(minX) * time.Second)
	maxT := chartBase.Add(time.Duration(maxX) * time.Second)

	c := timeserieslinechart.New(w, h,
		timeserieslinechart.WithXLabelFormatter(elapsedLabel),
		timeserieslinechart.WithYLabelFormatter(linechart.DefaultLabelFormatter()),
	)
	c.AxisStyle = axisStyle
	c.LabelStyle = labelStyle
	c.AutoMinY, c.AutoMaxY = false, false
	c.SetYRange(minY, maxY)
	c.SetViewYRange(minY, maxY)
	c.SetTimeRange(minT, maxT)
	c.SetViewTimeRange(minT, maxT)

	for _, s := range series {
		if len(s.points) == 0 {
			continue
		}
		c.SetDataSetStyle(s.name, s.style)
		for _, p := range s.points {
			c.PushDataSet(s.name, timeserieslinechart.TimePoint{
				Time:  chartBase.Add(time.Duration(p.X) * time.Second),
				Value: s.convValue(p.Y),
			})
		}
	}
	c.DrawBrailleAll()
	return c.View()
}

// xBounds returns the elapsed-seconds range across all series, widened to a
// minimum span. ok is false when there are no points.
func xBounds(series []chartSeries) (minX, maxX float64, ok bool) {
	seen := false
	for _, s := range series {
		for _, p := range s.points {
			if !seen {
				minX, maxX, seen = p.X, p.X, true
				continue
			}
			minX = min(minX, p.X)
			maxX = max(maxX, p.X)
		}
	}
	if !seen {
		return 0, 0, false
	}
	if maxX-minX < 1 {
		maxX = minX + 1
	}
	return minX, maxX, true
}

func waitingBox(w, h int) string {
	return lipgloss.NewStyle().
		Width(w).Height(h).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(colorDim).
		Render("waiting for data…")
}

// renderRoRDual draws the temperature series on the left axis with the RoR curve
// overlaid against an independent right axis, so both share the chart's full
// height instead of being stacked into half-height panels. rorPts are raw °C/min
// values; they are unit-converted, floored at 0 (negatives clip), mapped into the
// temperature coordinate space, and labeled on the right. Falls back to a plain
// temperature chart when there's no RoR to show.
func renderRoRDual(w, h int, temp []chartSeries, rorPts []canvas.Float64Point) string {
	if w < 8 || h < 4 {
		return ""
	}
	tLo, tHi, ok := yBounds(temp, nil)
	if !ok {
		return waitingBox(w, h)
	}

	// Convert RoR to the display unit and find its top (floored at 0).
	conv := make([]canvas.Float64Point, 0, len(rorPts))
	rHi := 0.0
	for _, p := range rorPts {
		v := convDelta(p.Y)
		conv = append(conv, canvas.Float64Point{X: p.X, Y: v})
		rHi = max(rHi, v)
	}
	if len(conv) == 0 || rHi <= 0 {
		return chartCore(w, h, temp, tLo, tHi)
	}
	rLo := 0.0
	rHi *= 1.1 // headroom above the peak

	// Map RoR into the temperature axis: r=rLo→tLo, r=rHi→tHi. Negatives land
	// below tLo and clip. The right axis then reads back the true RoR value.
	mapped := make([]canvas.Float64Point, len(conv))
	for i, p := range conv {
		mapped[i] = canvas.Float64Point{X: p.X, Y: tLo + (p.Y-rLo)/(rHi-rLo)*(tHi-tLo)}
	}
	series := append(append([]chartSeries{}, temp...),
		chartSeries{name: "ror", style: rorLineStyle, points: mapped, kind: valueRaw})

	axis := rightAxis(h, rLo, rHi, rorLineStyle)
	chart := chartCore(w-lipgloss.Width(axis), h, series, tLo, tHi)
	return lipgloss.JoinHorizontal(lipgloss.Top, chart, axis)
}

// rightAxis renders a right-hand Y-axis label strip h rows tall for the value
// range [lo,hi]. Labels sit every other row from the top graph row (hi) down to
// the axis baseline at row h-2 (lo); the final row aligns with the X labels and
// stays blank. Values are linear in row, matching how the chart maps its Y axis.
func rightAxis(h int, lo, hi float64, style lipgloss.Style) string {
	label := func(v float64) string { return fmt.Sprintf("%.0f", v) }
	w := len(label(hi))
	if n := len(label(lo)); n > w {
		w = n
	}
	w++ // gap from the plot

	lines := make([]string, h)
	for i := range lines {
		lines[i] = strings.Repeat(" ", w)
	}
	bottom := h - 2 // baseline row (row h-1 holds the X-axis labels)
	for row := 0; row <= bottom; row += 2 {
		v := hi - float64(row)/float64(bottom)*(hi-lo)
		lines[row] = style.Render(fmt.Sprintf("%*s", w, label(v)))
	}
	return strings.Join(lines, "\n")
}
