package roest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Datapoint is one time-series sample from a roast log.
type Datapoint struct {
	Msec     int      `json:"msec"`
	BT       *float64 `json:"bt"`
	ET       *float64 `json:"et"`
	Target   *float64 `json:"target"`
	Fan      *float64 `json:"fan"`
	Heat     *float64 `json:"heat"`
	RPM      *float64 `json:"rpm"`
	RorFloat *float64 `json:"ror_float"`
	Ror      *float64 `json:"ror"`
}

// RoR returns the Rate of Rise, preferring the precise float value.
func (d Datapoint) RoR() (float64, bool) {
	if d.RorFloat != nil {
		return *d.RorFloat, true
	}
	if d.Ror != nil {
		return *d.Ror, true
	}
	return 0, false
}

// GetDatapoints fetches every datapoint for a log in a single request using the
// page_size=all trick (the same call the web app uses to seed a roast chart).
func (c *Client) GetDatapoints(logID int) ([]Datapoint, error) {
	params := url.Values{}
	params.Set("log", strconv.Itoa(logID))
	params.Set("page_size", "all")
	body, err := c.get("/datapoints/", params)
	if err != nil {
		return nil, err
	}
	// With page_size=all the endpoint returns a bare array; without it (or when
	// empty) it can return the standard paginated envelope. Handle both.
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []Datapoint
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, fmt.Errorf("decoding datapoints: %w", err)
		}
		return list, nil
	}
	var env struct {
		Results []Datapoint `json:"results"`
	}
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return nil, fmt.Errorf("decoding datapoints: %w", err)
	}
	return env.Results, nil
}
