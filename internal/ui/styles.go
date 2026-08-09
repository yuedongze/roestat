package ui

import "github.com/charmbracelet/lipgloss"

// Palette. Warm tones nod to coffee roasting; blue/red distinguish the BT/ET
// curves in the charts.
var (
	colorAccent = lipgloss.Color("208") // orange
	colorBT     = lipgloss.Color("9")   // red    — bean temp
	colorET     = lipgloss.Color("12")  // blue   — env temp
	colorTarget = lipgloss.Color("2")   // green  — target
	colorRoR    = lipgloss.Color("13")  // magenta — rate of rise
	colorDim    = lipgloss.Color("245") // muted grey
	colorAxis   = lipgloss.Color("240")
	colorLabel  = lipgloss.Color("244")

	// Roast-phase bar: soft green → pale yellow → tan, mirroring the web app.
	colorPhaseDrying      = lipgloss.Color("150") // green  — drying
	colorPhaseMaillard    = lipgloss.Color("229") // yellow — Maillard
	colorPhaseDevelopment = lipgloss.Color("180") // tan    — development
	colorPhaseText        = lipgloss.Color("235") // dark ink on the light segments
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(colorAccent).
			Padding(0, 1)

	statLabelStyle = lipgloss.NewStyle().Foreground(colorDim)
	statValueStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAxis).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 1)

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)

	spinnerHintStyle = lipgloss.NewStyle().Foreground(colorDim)

	axisStyle  = lipgloss.NewStyle().Foreground(colorAxis)
	labelStyle = lipgloss.NewStyle().Foreground(colorLabel)

	btLineStyle     = lipgloss.NewStyle().Foreground(colorBT)
	etLineStyle     = lipgloss.NewStyle().Foreground(colorET)
	targetLineStyle = lipgloss.NewStyle().Foreground(colorTarget)
	rorLineStyle    = lipgloss.NewStyle().Foreground(colorRoR)

	legendBTStyle     = lipgloss.NewStyle().Foreground(colorBT).Bold(true)
	legendETStyle     = lipgloss.NewStyle().Foreground(colorET).Bold(true)
	legendTargetStyle = lipgloss.NewStyle().Foreground(colorTarget).Bold(true)
	legendRoRStyle    = lipgloss.NewStyle().Foreground(colorRoR).Bold(true)

	// Filled bar segments (dark text on a light background) and matching legend
	// swatches (foreground-only) for the roast-phase breakdown.
	phaseSegStyles = []lipgloss.Style{
		lipgloss.NewStyle().Background(colorPhaseDrying).Foreground(colorPhaseText),
		lipgloss.NewStyle().Background(colorPhaseMaillard).Foreground(colorPhaseText),
		lipgloss.NewStyle().Background(colorPhaseDevelopment).Foreground(colorPhaseText),
	}
	phaseLegendStyles = []lipgloss.Style{
		lipgloss.NewStyle().Foreground(colorPhaseDrying).Bold(true),
		lipgloss.NewStyle().Foreground(colorPhaseMaillard).Bold(true),
		lipgloss.NewStyle().Foreground(colorPhaseDevelopment).Bold(true),
	}
)
