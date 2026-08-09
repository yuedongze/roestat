package ui

import (
	"math"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yuedongze/roestat/internal/roest"
)

func ptr[T any](v T) *T { return &v }

// syntheticRoast builds a plausible S-curve roast for chart rendering.
func syntheticRoast() []roest.Datapoint {
	var pts []roest.Datapoint
	for s := 0; s <= 600; s += 2 {
		bt := 90 + 120*(1-math.Exp(-float64(s)/300))
		et := bt + 15
		tg := bt + 5
		ror := 60 * math.Exp(-float64(s)/300)
		pts = append(pts, roest.Datapoint{
			Msec: s * 1000, BT: ptr(bt), ET: ptr(et), Target: ptr(tg), RorFloat: ptr(ror),
		})
	}
	return pts
}

func liveDataAt(i int) liveDataMsg {
	sec := i
	bt := 90 + 1.5*float64(i)
	// Events accumulate as the roast passes milestones, so the phase bar grows.
	events := []roest.LiveEvent{{Msec: 0, Type: 0}} // charge
	if i >= 10 {
		events = append(events, roest.LiveEvent{Msec: 10000, Type: 4}) // dry end
	}
	if i >= 25 {
		events = append(events, roest.LiveEvent{Msec: 25000, Type: 5}) // first crack
	}
	return liveDataMsg(roest.LivePayload{
		BatchUUID: "999", ProfileID: "565318",
		Events: events,
		Data: roest.LiveData{
			Msec: sec * 1000, BT: bt, ET: bt + 12, Target: bt + 4,
			Fan: 81, Heat: 51, RPM: 60,
		},
	})
}

// TestRenderAllViews drives the model through every view with synthetic data and
// asserts each renders non-empty output without panicking. No network/TTY.
func TestRenderAllViews(t *testing.T) {
	var m tea.Model = NewApp(nil)
	send := func(msg tea.Msg) { m, _ = m.Update(msg) }

	send(tea.WindowSizeMsg{Width: 100, Height: 30})

	bean := "Ethiopia Guji"
	send(logsPageMsg{
		logs: []roest.Log{
			{ID: 1, BatchNo: 1823, BeanName: &bean, StartWeight: ptr(600.0), EndWeight: ptr(510.0),
				FCTemp: ptr(196.0), StartTimestamp: ptr("2026-08-08T10:00:00Z"), EndTimestamp: ptr("2026-08-08T10:11:00Z"),
				DryEndMsec: ptr(305000), FirstCrackMsec: ptr(530000)},
			{ID: 2, BatchNo: 1824},
		},
		page: 1, hasNext: true,
	})
	assertContains(t, "history", m.View(), "Ethiopia Guji", "1823")

	// Detail view.
	send(tea.KeyMsg{Type: tea.KeyEnter})
	send(datapointsMsg{logID: 1, points: syntheticRoast()})
	assertContains(t, "detail", m.View(), "Batch #1823", "BT", "RoR",
		"Drying", "Maillard", "Development") // phase bar

	// Back to history, then to the picker.
	send(tea.KeyMsg{Type: tea.KeyEsc})
	send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	send(machinesMsg{machines: []roest.Machine{{ID: 2483, Name: "Neset", MachineID: "abc"}}})
	assertContains(t, "picker", m.View(), "Neset")

	// Select machine -> live view, then stream data.
	send(tea.KeyMsg{Type: tea.KeyEnter})
	send(liveConnectedMsg{machine: roest.Machine{ID: 2483, Name: "Neset"}, client: nil})
	for i := range 40 {
		send(liveDataAt(i))
	}
	assertContains(t, "live", m.View(), "LIVE", "Neset", "BT",
		"Drying", "Maillard", "Development") // growing phase bar

	t.Logf("\n%s", m.View()) // visible with -v
}

func assertContains(t *testing.T, view, got string, wants ...string) {
	t.Helper()
	if strings.TrimSpace(got) == "" {
		t.Fatalf("%s view rendered empty", view)
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("%s view missing %q\n---\n%s\n---", view, w, got)
		}
	}
}
