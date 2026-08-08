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

// Log is a single roast-session record. Optional fields are pointers because
// the API returns them as null for incomplete or unnamed roasts.
type Log struct {
	ID             int          `json:"id"`
	BatchNo        int          `json:"batch_no"`
	BeanName       *string      `json:"bean_name"`
	MachineSlug    string       `json:"machine_slug"`
	StartTimestamp *string      `json:"start_timestamp"`
	EndTimestamp   *string      `json:"end_timestamp"`
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
