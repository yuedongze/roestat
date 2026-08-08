// Package roest is a client for the ROEST coffee-roaster API
// (https://api.roestcoffee.com). It ports the integration proven out by the
// project's Python scripts: OAuth token auth, the /logs/ and /datapoints/
// endpoints, /machines/ with MQTT credentials, and the live MQTT feed.
package roest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// BaseURL is the ROEST REST API root (no trailing slash; endpoints add it).
const BaseURL = "https://api.roestcoffee.com"

// Client talks to the ROEST REST API with a bearer token obtained via the
// OAuth2 client-credentials grant.
type Client struct {
	http       *http.Client
	token      string
	customerID int
}

// NewClient reads ROEST_CLIENT_ID / ROEST_CLIENT_SECRET from the environment,
// exchanges them for an access token, and resolves the customer ID (from
// ROEST_CUSTOMER_ID if set, otherwise auto-detected from the account).
func NewClient() (*Client, error) {
	id := os.Getenv("ROEST_CLIENT_ID")
	secret := os.Getenv("ROEST_CLIENT_SECRET")
	if id == "" || secret == "" {
		return nil, fmt.Errorf("ROEST_CLIENT_ID and ROEST_CLIENT_SECRET must be set")
	}
	hc := &http.Client{Timeout: 30 * time.Second}
	token, err := fetchToken(hc, id, secret)
	if err != nil {
		return nil, err
	}
	c := &Client{http: hc, token: token}

	cust, err := resolveCustomerID(c)
	if err != nil {
		return nil, err
	}
	c.customerID = cust
	return c, nil
}

// CustomerID returns the resolved customer ID.
func (c *Client) CustomerID() int { return c.customerID }

// resolveCustomerID uses ROEST_CUSTOMER_ID when set, otherwise reads the
// authenticated account's first customer from /users/.
func resolveCustomerID(c *Client) (int, error) {
	if v := os.Getenv("ROEST_CUSTOMER_ID"); v != "" {
		id, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, fmt.Errorf("invalid ROEST_CUSTOMER_ID %q: %w", v, err)
		}
		return id, nil
	}

	body, err := c.get("/users/", nil)
	if err != nil {
		return 0, fmt.Errorf("resolving customer id: %w (set ROEST_CUSTOMER_ID to skip)", err)
	}
	// /users/ may return the user object directly or wrapped in a list/envelope.
	type user struct {
		Customers []struct {
			ID int `json:"id"`
		} `json:"customers"`
	}
	var u user
	if err := json.Unmarshal(body, &u); err != nil {
		var list []user
		if err2 := json.Unmarshal(body, &list); err2 == nil && len(list) > 0 {
			u = list[0]
		} else {
			var env struct {
				Results []user `json:"results"`
			}
			if err3 := json.Unmarshal(body, &env); err3 == nil && len(env.Results) > 0 {
				u = env.Results[0]
			}
		}
	}
	if len(u.Customers) == 0 {
		return 0, fmt.Errorf("no customer found for this account; set ROEST_CUSTOMER_ID")
	}
	return u.Customers[0].ID, nil
}

func fetchToken(hc *http.Client, id, secret string) (string, error) {
	// Note: the token endpoint expects a JSON body, not form encoding.
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     id,
		"client_secret": secret,
	})
	resp, err := hc.Post(BaseURL+"/o/token/", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request returned %s: %s", resp.Status, truncate(b, 200))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(sanitizeJSON(b), &tr); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("token response contained no access_token")
	}
	return tr.AccessToken, nil
}

// get performs an authenticated GET and returns the sanitized response body.
func (c *Client) get(path string, params url.Values) ([]byte, error) {
	u := BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.getURL(u)
}

// getURL fetches a fully-formed URL (used to follow pagination "next" links,
// whose query params are already embedded).
func (c *Client) getURL(u string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s returned %s: %s", u, resp.Status, truncate(b, 200))
	}
	return sanitizeJSON(b), nil
}

// sanitizeJSON replaces raw control characters (< 0x20) that appear inside JSON
// string literals with spaces. The ROEST API emits unescaped control chars in
// free-text fields (bean names, notes); Go's encoding/json rejects them, so we
// mirror the Python scripts' json.loads(..., strict=False) leniency.
func sanitizeJSON(b []byte) []byte {
	out := make([]byte, 0, len(b))
	inString := false
	escaped := false
	for _, c := range b {
		if inString {
			switch {
			case escaped:
				escaped = false
				out = append(out, c)
			case c == '\\':
				escaped = true
				out = append(out, c)
			case c == '"':
				inString = false
				out = append(out, c)
			case c < 0x20:
				out = append(out, ' ') // strip the raw control char
			default:
				out = append(out, c)
			}
			continue
		}
		if c == '"' {
			inString = true
		}
		out = append(out, c)
	}
	return out
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}
