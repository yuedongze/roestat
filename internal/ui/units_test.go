package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yuedongze/roestat/internal/roest"
)

func TestTempConversions(t *testing.T) {
	defer func() { currentUnit = unitCelsius }()

	currentUnit = unitCelsius
	if convTemp(100) != 100 || convDelta(10) != 10 || tempSuffix() != "°C" || rorSuffix() != "°C/min" {
		t.Fatal("celsius should be identity")
	}

	currentUnit = unitFahrenheit
	if convTemp(0) != 32 || convTemp(100) != 212 {
		t.Errorf("convTemp F: 0->%.1f 100->%.1f, want 32/212", convTemp(0), convTemp(100))
	}
	if convDelta(10) != 18 { // a rate has no offset
		t.Errorf("convDelta F: 10->%.1f, want 18", convDelta(10))
	}
	if tempSuffix() != "°F" || rorSuffix() != "°F/min" {
		t.Error("fahrenheit suffixes wrong")
	}

	toggleUnit()
	if currentUnit != unitCelsius {
		t.Error("toggleUnit should flip F back to C")
	}
}

// TestFahrenheitToggle drives the app and checks the "f" key reunits the views.
func TestFahrenheitToggle(t *testing.T) {
	defer func() { currentUnit = unitCelsius }()
	currentUnit = unitCelsius

	var m tea.Model = NewApp(nil)
	send := func(msg tea.Msg) { m, _ = m.Update(msg) }
	send(tea.WindowSizeMsg{Width: 100, Height: 30})
	send(logsPageMsg{
		logs: []roest.Log{{ID: 1, BatchNo: 1823, FCTemp: ptr(196.0),
			StartTimestamp: ptr("2026-08-08T10:00:00Z"), EndTimestamp: ptr("2026-08-08T10:11:00Z")}},
		page: 1,
	})

	// History FC column header reflects the unit.
	if v := m.View(); !strings.Contains(v, "FC °C") {
		t.Fatalf("history header not in °C:\n%s", v)
	}

	// Detail view starts in Celsius (check the body, not the footer key hint).
	send(tea.KeyMsg{Type: tea.KeyEnter})
	send(datapointsMsg{logID: 1, points: syntheticRoast()})
	body := m.(App).detail.view()
	if !strings.Contains(body, "°C") || strings.Contains(body, "°F") {
		t.Fatalf("detail not purely celsius:\n%s", body)
	}

	// Toggle to Fahrenheit — the whole detail view reunits, FC 196°C -> 385°F.
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("f")})
	body = m.(App).detail.view()
	if !strings.Contains(body, "°F") || strings.Contains(body, "°C") {
		t.Fatalf("detail not purely fahrenheit after toggle:\n%s", body)
	}
	if !strings.Contains(body, "385") {
		t.Errorf("expected converted FC 385°F in detail:\n%s", body)
	}

	// History rows were rebuilt too.
	if !strings.Contains(m.(App).history.view(), "FC °F") {
		t.Error("history header did not switch to °F")
	}
}
