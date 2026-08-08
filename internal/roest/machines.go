package roest

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// MQTTConfig holds the per-machine credentials for the live MQTT feed.
type MQTTConfig struct {
	Username          string `json:"username"`
	SubscribePassword string `json:"subscribe_password"`
	Topic             string `json:"topic"`
}

// Machine is a roasting machine registered to the customer.
type Machine struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	MachineID  string     `json:"machine_id"`
	MQTTConfig MQTTConfig `json:"mqtt_config"`
}

// GetMachines lists the customer's machines. The endpoint may return either a
// paginated envelope ({"results": [...]}) or a bare array, so we handle both.
func (c *Client) GetMachines() ([]Machine, error) {
	body, err := c.get("/machines/", nil)
	if err != nil {
		return nil, err
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var list []Machine
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, fmt.Errorf("decoding machines: %w", err)
		}
		return list, nil
	}
	var env struct {
		Results []Machine `json:"results"`
	}
	if err := json.Unmarshal(trimmed, &env); err != nil {
		return nil, fmt.Errorf("decoding machines: %w", err)
	}
	return env.Results, nil
}
