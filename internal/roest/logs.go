package roest

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ProfileData is the inline profile summary embedded in a Log.
type ProfileData struct {
	Name string `json:"name"`
}

// LogEvent is a roast milestone recorded on a log, in elapsed msec from charge
// (type 0 = charge, 1 = drop, 4 = dry end, 5 = first crack).
type LogEvent struct {
	Msec int `json:"msec"`
	Type int `json:"type"`
}

// RoastPhase is one segment of a roast: Drying (charge→dry end), Maillard (dry
// end→first crack) or Development (first crack→drop).
type RoastPhase struct {
	Name     string
	Duration time.Duration
	Fraction float64 // share of the total roast time (charge→drop)
}

// Log is a single roast-session record. Optional fields are pointers because
// the API returns them as null for incomplete or unnamed roasts.
type Log struct {
	ID             int          `json:"id"`
	BatchNo        int          `json:"batch_no"`
	BeanName       *string      `json:"bean_name"`
	MachineSlug    string       `json:"machine_slug"`
	StartTimestamp *string      `json:"start_timestamp"`
	EndTimestamp   *string      `json:"end_timestamp"`
	DropTimestamp  *string      `json:"drop_timestamp"`
	DryEndMsec     *int         `json:"dryend_event_msec"`
	FirstCrackMsec *int         `json:"firstcrack_event_msec"`
	Events         []LogEvent   `json:"events"`
	StartWeight    *float64     `json:"start_weight"`
	EndWeight      *float64     `json:"end_weight"`
	FCTemp         *float64     `json:"fc_temp"`
	EndTemp        *float64     `json:"end_temp"`
	WholeBeanColor *float64     `json:"whole_bean_color"`
	ProfileName    *string      `json:"profile_name"`
	ProfileData    *ProfileData `json:"profile_data"`
}

// Bean returns the bean name, or "Unnamed" when absent.
func (l Log) Bean() string {
	if l.BeanName != nil && *l.BeanName != "" {
		return *l.BeanName
	}
	return "Unnamed"
}

// Profile returns the profile name from either the inline data or the override.
func (l Log) Profile() string {
	if l.ProfileData != nil && l.ProfileData.Name != "" {
		return l.ProfileData.Name
	}
	if l.ProfileName != nil {
		return *l.ProfileName
	}
	return "—"
}

// StartTime parses the roast start timestamp.
func (l Log) StartTime() (time.Time, bool) { return parseTS(l.StartTimestamp) }

// Duration is the wall-clock roast length, if both timestamps are present.
func (l Log) Duration() (time.Duration, bool) {
	start, ok := parseTS(l.StartTimestamp)
	if !ok {
		return 0, false
	}
	end, ok := parseTS(l.EndTimestamp)
	if !ok {
		return 0, false
	}
	return end.Sub(start), true
}

// Phases splits the roast into Drying / Maillard / Development segments with
// their durations and shares of total roast time. ok is false when the boundary
// events (dry end, first crack, drop) aren't all available or aren't ordered.
func (l Log) Phases() ([]RoastPhase, bool) {
	dryEnd, ok1 := l.eventMsec(4, l.DryEndMsec)
	firstCrack, ok2 := l.eventMsec(5, l.FirstCrackMsec)
	drop, ok3 := l.dropMsec()
	if !ok1 || !ok2 || !ok3 {
		return nil, false
	}
	// Boundaries must be strictly increasing for the segments to make sense.
	if dryEnd <= 0 || firstCrack <= dryEnd || drop <= firstCrack {
		return nil, false
	}
	return buildPhases(dryEnd, firstCrack, drop, true, true)
}

// buildPhases assembles the ordered Drying/Maillard/Development segments that
// have occurred by time end, using end as the denominator for the fractions.
// Milestones that haven't happened (or fall at/after end) are omitted, so a
// roast still in its drying phase yields a single growing segment. ok is false
// only when end is non-positive.
func buildPhases(dryEnd, firstCrack, end int, haveDryEnd, haveFirstCrack bool) ([]RoastPhase, bool) {
	if end <= 0 {
		return nil, false
	}
	type bound struct {
		name  string
		start int
	}
	bounds := []bound{{"Drying", 0}}
	if haveDryEnd && dryEnd > 0 && dryEnd < end {
		bounds = append(bounds, bound{"Maillard", dryEnd})
	}
	if haveFirstCrack && firstCrack < end && firstCrack > bounds[len(bounds)-1].start {
		bounds = append(bounds, bound{"Development", firstCrack})
	}

	total := float64(end)
	phases := make([]RoastPhase, len(bounds))
	for i, b := range bounds {
		to := end
		if i+1 < len(bounds) {
			to = bounds[i+1].start
		}
		phases[i] = RoastPhase{
			Name:     b.name,
			Duration: time.Duration(to-b.start) * time.Millisecond,
			Fraction: float64(to-b.start) / total,
		}
	}
	return phases, true
}

// eventMsec returns the elapsed-msec of an event, preferring the dedicated log
// field and falling back to the events array.
func (l Log) eventMsec(typ int, direct *int) (int, bool) {
	if direct != nil && *direct > 0 {
		return *direct, true
	}
	for _, e := range l.Events {
		if e.Type == typ && e.Msec > 0 {
			return e.Msec, true
		}
	}
	return 0, false
}

// dropMsec returns the drop time (total roast length) in elapsed msec, from the
// drop event, else the drop timestamp, else the overall duration.
func (l Log) dropMsec() (int, bool) {
	for _, e := range l.Events {
		if e.Type == 1 && e.Msec > 0 {
			return e.Msec, true
		}
	}
	if start, ok := parseTS(l.StartTimestamp); ok {
		if drop, ok := parseTS(l.DropTimestamp); ok {
			return int(drop.Sub(start).Milliseconds()), true
		}
	}
	if d, ok := l.Duration(); ok {
		return int(d.Milliseconds()), true
	}
	return 0, false
}

// WeightLossPct is the roast weight loss as a percentage.
func (l Log) WeightLossPct() (float64, bool) {
	if l.StartWeight == nil || l.EndWeight == nil || *l.StartWeight == 0 {
		return 0, false
	}
	return (*l.StartWeight - *l.EndWeight) / *l.StartWeight * 100, true
}

func parseTS(s *string) (time.Time, bool) {
	if s == nil || *s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

type logsEnvelope struct {
	Count   int     `json:"count"`
	Next    *string `json:"next"`
	Results []Log   `json:"results"`
}

// activeLogMatchTolerance bounds how far an in-progress log's start may sit from
// the live charge time before we consider it a match.
const activeLogMatchTolerance = 120 * time.Second

// FindActiveLog returns the in-progress log (EndTimestamp == nil) that best
// matches the given machine and charge time, so the live view can backfill the
// datapoints collected before it connected. found is false when nothing looks
// like an active roast.
//
// The live payload carries no numeric log ID (and Log has no UUID), so we
// correlate by start time: among unfinished logs on the newest page, we pick the
// one whose StartTimestamp is closest to chargeUnix, preferring a matching
// machine slug and falling back to the newest unfinished log.
func (c *Client) FindActiveLog(m Machine, chargeUnix int64) (Log, bool, error) {
	logs, _, err := c.GetLogsPage(1)
	if err != nil {
		return Log{}, false, err
	}

	charge := time.Unix(chargeUnix, 0)

	// timed: closest unfinished log whose start is within tolerance of charge.
	// fallback: newest unfinished log, used when no start time correlates.
	timed, fallback := -1, -1
	var bestDelta time.Duration
	var fallbackStart time.Time
	for i, l := range logs {
		if l.EndTimestamp != nil { // finished roast
			continue
		}
		start, hasStart := l.StartTime()

		if chargeUnix > 0 && hasStart {
			delta := start.Sub(charge)
			if delta < 0 {
				delta = -delta
			}
			if delta <= activeLogMatchTolerance {
				// Keep the closest match, breaking ties toward this machine.
				if timed < 0 || delta < bestDelta ||
					(delta == bestDelta && l.MachineSlug == m.MachineID) {
					timed, bestDelta = i, delta
				}
				continue
			}
		}

		// Track the newest unfinished log as a fallback.
		if fallback < 0 || (hasStart && start.After(fallbackStart)) {
			fallback, fallbackStart = i, start
		}
	}

	switch {
	case timed >= 0:
		return logs[timed], true, nil
	case fallback >= 0:
		return logs[fallback], true, nil
	default:
		return Log{}, false, nil
	}
}

// GetLogsPage fetches one page (25 logs) of the customer's roast history.
// page is 1-based; it reports whether a further page exists.
func (c *Client) GetLogsPage(page int) (logs []Log, hasNext bool, err error) {
	params := url.Values{}
	params.Set("customer", strconv.Itoa(c.customerID))
	if page > 1 {
		params.Set("page", strconv.Itoa(page))
	}
	body, err := c.get("/logs/", params)
	if err != nil {
		return nil, false, err
	}
	var env logsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, false, fmt.Errorf("decoding logs: %w", err)
	}
	return env.Results, env.Next != nil, nil
}
