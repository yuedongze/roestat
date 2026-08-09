package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/yuedongze/roestat/internal/roest"
)

// statLine renders "Label value   Label value …" on one line.
func statLine(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = statLabelStyle.Render(p[0]+" ") + statValueStyle.Render(p[1])
	}
	return strings.Join(parts, statLabelStyle.Render("   "))
}

// statGrid renders label/value pairs as a wrapped multi-column block.
func statGrid(pairs [][2]string, cols int) string {
	if cols < 1 {
		cols = 1
	}
	var rows []string
	for i := 0; i < len(pairs); i += cols {
		end := min(i+cols, len(pairs))
		rows = append(rows, statLine(pairs[i:end]))
	}
	return strings.Join(rows, "\n")
}

// tempLegend labels the BT/ET/Target curves with their colors.
func tempLegend() string {
	return legendBTStyle.Render("■ BT") + "  " +
		legendETStyle.Render("■ ET") + "  " +
		legendTargetStyle.Render("■ Target")
}

func joinVertical(blocks ...string) string {
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// phaseLegend labels the roast-phase segments with their colors.
func phaseLegend() string {
	names := []string{"Drying", "Maillard", "Development"}
	parts := make([]string, len(names))
	for i, n := range names {
		parts[i] = phaseLegendStyles[i%len(phaseLegendStyles)].Render("■ " + n)
	}
	return strings.Join(parts, "  ")
}

// phaseBar renders the roast phases as a full-width stacked bar, each segment
// colored and labeled with its duration and share of the roast.
func phaseBar(width int, phases []roest.RoastPhase) string {
	if width < 6 || len(phases) == 0 {
		return ""
	}

	// Allocate integer column widths proportional to each phase's share, handing
	// the rounding remainder to the segments with the largest fractional parts so
	// the bar exactly fills the width.
	widths := make([]int, len(phases))
	rem := make([]float64, len(phases))
	used := 0
	for i, p := range phases {
		exact := p.Fraction * float64(width)
		widths[i] = int(exact)
		rem[i] = exact - float64(widths[i])
		used += widths[i]
	}
	for used < width {
		best := 0
		for i := range rem {
			if rem[i] > rem[best] {
				best = i
			}
		}
		widths[best]++
		rem[best] = -1
		used++
	}

	segs := make([]string, len(phases))
	for i, p := range phases {
		style := phaseSegStyles[i%len(phaseSegStyles)]
		segs[i] = style.Width(widths[i]).MaxWidth(widths[i]).MaxHeight(1).
			Align(lipgloss.Center).Render(phaseLabel(p, widths[i]))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, segs...)
}

// phaseLabel formats a phase as "m:ss (pct%)", shrinking to just the percentage
// when the segment is too narrow for the full label.
func phaseLabel(p roest.RoastPhase, w int) string {
	mins := int(p.Duration.Minutes())
	secs := int(p.Duration.Seconds()) % 60
	label := fmt.Sprintf("%d:%02d (%.1f%%)", mins, secs, p.Fraction*100)
	if lipgloss.Width(label) > w {
		label = fmt.Sprintf("%.0f%%", p.Fraction*100)
	}
	return label
}
