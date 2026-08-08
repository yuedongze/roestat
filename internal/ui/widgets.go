package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
