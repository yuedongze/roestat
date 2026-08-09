package ui

import (
	"testing"

	"github.com/yuedongze/roestat/internal/roest"
)

// backfillPoints builds REST datapoints at 1s spacing from 0..maxSec inclusive.
func backfillPoints(maxSec int) []roest.Datapoint {
	var pts []roest.Datapoint
	for s := 0; s <= maxSec; s++ {
		bt := 100 + float64(s)
		pts = append(pts, roest.Datapoint{
			Msec: s * 1000, BT: ptr(bt), ET: ptr(bt + 10), Target: ptr(bt + 5),
		})
	}
	return pts
}

// liveMsecs returns the elapsed-second X values of the model's bean-temp series.
func liveMsecs(m *liveModel) []int {
	secs := make([]int, len(m.btPts))
	for i, p := range m.btPts {
		secs[i] = int(p.X)
	}
	return secs
}

func assertAscending(t *testing.T, m *liveModel) {
	t.Helper()
	for i := 1; i < len(m.btPts); i++ {
		if m.btPts[i].X <= m.btPts[i-1].X {
			t.Fatalf("btPts not strictly ascending at %d: %v", i, liveMsecs(m))
		}
	}
	// The four live slices must stay index-aligned and equal length.
	if len(m.etPts) != len(m.btPts) || len(m.tgPts) != len(m.btPts) || len(m.samples) != len(m.btPts) {
		t.Fatalf("series lengths diverged: bt=%d et=%d tg=%d samples=%d",
			len(m.btPts), len(m.etPts), len(m.tgPts), len(m.samples))
	}
}

func TestBackfillBeforeLive(t *testing.T) {
	m := newLive()
	m.begin(roest.Machine{ID: 1})

	m.applyBackfill(backfillMsg{found: true, points: backfillPoints(10)}) // 0..10s
	if m.backfillMax != 10000 {
		t.Fatalf("backfillMax = %d, want 10000", m.backfillMax)
	}

	// Live samples: 8s and 9s fall inside the backfill and must be dropped; 11s..13s append.
	for _, sec := range []int{8, 9, 11, 12, 13} {
		m.ingest(roest.LivePayload{Data: roest.LiveData{Msec: sec * 1000, BT: 200, ET: 210, Target: 205}})
	}

	got := liveMsecs(&m)
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13}
	if len(got) != len(want) {
		t.Fatalf("merged seconds = %v, want %v", got, want)
	}
	assertAscending(t, &m)
}

func TestLiveBeforeBackfill(t *testing.T) {
	m := newLive()
	m.begin(roest.Machine{ID: 1})

	// Live arrives first, overlapping the range REST will later cover, plus a tail.
	for _, sec := range []int{7, 8, 9, 11, 12} {
		m.ingest(roest.LivePayload{Data: roest.LiveData{Msec: sec * 1000, BT: 200, ET: 210, Target: 205}})
	}

	m.applyBackfill(backfillMsg{found: true, points: backfillPoints(10)}) // 0..10s

	// Overlap (7,8,9) dropped in favor of backfill; tail (11,12) preserved.
	got := liveMsecs(&m)
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	if len(got) != len(want) {
		t.Fatalf("merged seconds = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("merged seconds = %v, want %v", got, want)
		}
	}
	assertAscending(t, &m)
}

func TestBackfillNotFoundStaysLiveOnly(t *testing.T) {
	m := newLive()
	m.begin(roest.Machine{ID: 1})

	for _, sec := range []int{0, 1, 2} {
		m.ingest(roest.LivePayload{Data: roest.LiveData{Msec: sec * 1000, BT: 200, ET: 210, Target: 205}})
	}

	m.applyBackfill(backfillMsg{found: false})
	if m.backfillMax != -1 {
		t.Fatalf("backfillMax = %d, want -1 (no backfill applied)", m.backfillMax)
	}
	if len(m.btPts) != 3 {
		t.Fatalf("live-only series changed: %v", liveMsecs(&m))
	}
}
