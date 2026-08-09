package roest

import (
	"math"
	"testing"
)

func iptr(v int) *int       { return &v }
func sptr(v string) *string { return &v }

func TestPhasesFromEventMsecFields(t *testing.T) {
	// Drop time falls back to the roast duration (start→end) when no drop event.
	l := Log{
		StartTimestamp: sptr("2026-08-08T10:00:00Z"),
		EndTimestamp:   sptr("2026-08-08T10:11:00Z"), // 660s total
		DryEndMsec:     iptr(305000),                 // 5:05
		FirstCrackMsec: iptr(530000),                 // 8:50
	}
	phases, ok := l.Phases()
	if !ok {
		t.Fatal("Phases() = false, want true")
	}
	if len(phases) != 3 {
		t.Fatalf("got %d phases, want 3", len(phases))
	}

	want := []struct {
		name string
		secs float64
		frac float64
	}{
		{"Drying", 305, 305.0 / 660},
		{"Maillard", 225, 225.0 / 660},
		{"Development", 130, 130.0 / 660},
	}
	var totalFrac float64
	for i, w := range want {
		if phases[i].Name != w.name {
			t.Errorf("phase %d name = %q, want %q", i, phases[i].Name, w.name)
		}
		if got := phases[i].Duration.Seconds(); got != w.secs {
			t.Errorf("%s duration = %gs, want %gs", w.name, got, w.secs)
		}
		if got := phases[i].Fraction; math.Abs(got-w.frac) > 1e-9 {
			t.Errorf("%s fraction = %g, want %g", w.name, got, w.frac)
		}
		totalFrac += phases[i].Fraction
	}
	if math.Abs(totalFrac-1) > 1e-9 {
		t.Errorf("fractions sum to %g, want 1", totalFrac)
	}
}

func TestPhasesFromEventsArrayAndDropEvent(t *testing.T) {
	// No dedicated msec fields: everything comes from the events array, including
	// the drop event (type 1) which sets the total.
	l := Log{
		Events: []LogEvent{
			{Msec: 0, Type: 0},      // charge
			{Msec: 200000, Type: 4}, // dry end
			{Msec: 400000, Type: 5}, // first crack
			{Msec: 500000, Type: 1}, // drop
		},
	}
	phases, ok := l.Phases()
	if !ok {
		t.Fatal("Phases() = false, want true")
	}
	if phases[2].Fraction != 100000.0/500000 {
		t.Errorf("development fraction = %g, want 0.2", phases[2].Fraction)
	}
}

func TestLivePhasesGrows(t *testing.T) {
	charge := []LiveEvent{{Msec: 0, Type: 0}}
	dryEnd := append(charge, LiveEvent{Msec: 180000, Type: 4})
	firstCrack := append(dryEnd, LiveEvent{Msec: 300000, Type: 5})

	// Still drying: one segment spanning all elapsed time.
	if p, ok := LivePhases(charge, 120000); !ok || len(p) != 1 || p[0].Name != "Drying" || p[0].Fraction != 1 {
		t.Fatalf("drying-only: got %+v ok=%v", p, ok)
	}

	// After dry end, the Maillard segment grows against the elapsed total.
	p, ok := LivePhases(dryEnd, 300000)
	if !ok || len(p) != 2 {
		t.Fatalf("post dry-end: got %d phases ok=%v", len(p), ok)
	}
	if p[0].Fraction != 180000.0/300000 || p[1].Name != "Maillard" {
		t.Errorf("post dry-end fractions wrong: %+v", p)
	}

	// After first crack, all three phases show, development running up to now.
	if p, ok := LivePhases(firstCrack, 360000); !ok || len(p) != 3 || p[2].Name != "Development" {
		t.Fatalf("post first-crack: got %+v ok=%v", p, ok)
	}

	// A drop event caps the total even if a later sample arrives.
	dropped := append(firstCrack, LiveEvent{Msec: 360000, Type: 1})
	if p, ok := LivePhases(dropped, 999000); !ok || p[2].Fraction != 60000.0/360000 {
		t.Errorf("dropped: development fraction = %v (ok=%v), want %v", p, ok, 60000.0/360000)
	}

	// No elapsed time yet.
	if _, ok := LivePhases(charge, 0); ok {
		t.Error("zero elapsed: ok = true, want false")
	}
}

func TestPhasesUnavailable(t *testing.T) {
	cases := map[string]Log{
		"no data":            {},
		"missing firstcrack": {DryEndMsec: iptr(100000), Events: []LogEvent{{Msec: 300000, Type: 1}}},
		"out of order":       {DryEndMsec: iptr(300000), FirstCrackMsec: iptr(100000), EndTimestamp: sptr("2026-08-08T10:10:00Z"), StartTimestamp: sptr("2026-08-08T10:00:00Z")},
	}
	for name, l := range cases {
		if _, ok := l.Phases(); ok {
			t.Errorf("%s: Phases() = true, want false", name)
		}
	}
}
