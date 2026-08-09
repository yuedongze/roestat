package ui

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/linechart"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
)

// chartBase anchors the time axis: point X (elapsed seconds) is mapped to
// chartBase + X seconds, so the axis float value equals elapsed seconds.
var chartBase = time.Unix(0, 0).UTC()

// chartSeries is one named line to draw (X = elapsed seconds, Y = value).
type chartSeries struct {
	name   string
	style  lipgloss.Style
	points []canvas.Float64Point
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

	minX, maxX := 0.0, 1.0
	minY, maxY := 0.0, 1.0
	seen := false
	for _, s := range series {
		for _, p := range s.points {
			if !seen {
				minX, maxX, minY, maxY = p.X, p.X, p.Y, p.Y
				seen = true
				continue
			}
			minX = min(minX, p.X)
			maxX = max(maxX, p.X)
			minY = min(minY, p.Y)
			maxY = max(maxY, p.Y)
		}
	}
	if !seen {
		return lipgloss.NewStyle().
			Width(w).Height(h).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorDim).
			Render("waiting for data…")
	}

	// Pad the Y range a little and guard against zero-width ranges.
	pad := max((maxY-minY)*0.1, 1)
	minY -= pad
	maxY += pad
	// Optionally clamp the visible floor (e.g. hide negative RoR) and keep a
	// non-empty range if every point sits at/below the floor.
	if cfg.yFloor != nil && minY < *cfg.yFloor {
		minY = *cfg.yFloor
		if maxY <= minY {
			maxY = minY + 1
		}
	}
	if maxX-minX < 1 {
		maxX = minX + 1
	}

	minT := chartBase.Add(time.Duration(minX) * time.Second)
	maxT := chartBase.Add(time.Duration(maxX) * time.Second)

	c := timeserieslinechart.New(w, h,
		timeserieslinechart.WithXLabelFormatter(elapsedLabel),
		timeserieslinechart.WithYLabelFormatter(linechart.DefaultLabelFormatter()),
	)
	c.AxisStyle = axisStyle
	c.LabelStyle = labelStyle
	// The chart auto-expands its Y range to fit pushed points; with a floor set
	// that would drag the minimum back below it (e.g. down to a large negative
	// RoR), so pin the minimum and let below-floor points clip instead.
	if cfg.yFloor != nil {
		c.AutoMinY = false
	}
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
				Value: p.Y,
			})
		}
	}
	c.DrawBrailleAll()
	return c.View()
}
