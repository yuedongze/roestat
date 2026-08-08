package roest

// DefaultRoRWindowSec is the look-back window for Rate-of-Rise computation,
// matching the project's analyze_ror_v2.py (30s).
const DefaultRoRWindowSec = 30

// Sample is a minimal (time, bean-temp) pair for RoR computation.
type Sample struct {
	Msec int
	BT   float64
}

// RoR computes the Rate of Rise (°C/min) for the most recent sample, using the
// earliest sample at least windowSec back (falling back to the oldest sample).
// The live MQTT feed omits RoR, so we derive it ourselves.
func RoR(samples []Sample, windowSec int) (float64, bool) {
	if len(samples) < 2 {
		return 0, false
	}
	last := samples[len(samples)-1]
	target := last.Msec - windowSec*1000
	prev := samples[0]
	for i := len(samples) - 2; i >= 0; i-- {
		if samples[i].Msec <= target {
			prev = samples[i]
			break
		}
	}
	dtMin := float64(last.Msec-prev.Msec) / 1000 / 60
	if dtMin <= 0 {
		return 0, false
	}
	return (last.BT - prev.BT) / dtMin, true
}
