// Package homeassistant queries a Home Assistant instance's REST API,
// grouped by area (Upstairs / Downstairs / Bedroom), and emits a
// pipe-separated "<presence>|<upstairs>|<downstairs>|<bedroom>" string
// for the homeassistant scene — each area field already combines that
// area's climate, occupancy, and lights-on state into one line.
package homeassistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// area bundles the entities that make up one grouped row: a climate
// entity for the temperature reading, an occupancy sensor, and the
// individual lights physically in that area (whose on/off states get
// folded into one "lights on" flag for the row).
type area struct {
	name        string
	climate     string
	occupancy   string
	lightGroup  []string
}

// Client hits a fixed Home Assistant instance's REST API for a fixed set
// of entities. One http.Client with a 10s timeout is reused across Fetch
// calls.
type Client struct {
	baseURL string
	token   string
	http    *http.Client

	presenceEntity string
	areas          []area
}

// New builds a Client against baseURL (e.g. "http://10.0.0.7:8123") using
// token as a Home Assistant long-lived access token (profile > Security >
// Long-Lived Access Tokens in the HA UI). The entity set is fixed to this
// home's presence group and three areas — not configurable via env, same
// tradeoff as the hnKeywords list in serve.go.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		token:          token,
		http:           &http.Client{Timeout: 10 * time.Second},
		presenceEntity: "group.home_presence",
		areas: []area{
			{
				name:      "Upstairs",
				climate:   "climate.upstairs",
				occupancy: "binary_sensor.upstairs_occupancy_group",
				lightGroup: []string{
					"light.kitchen_ceiling_lights",
					"light.entryway_ceiling_lights",
					"light.upstairs_bathroom",
				},
			},
			{
				name:      "Downstairs",
				climate:   "climate.downstairs",
				occupancy: "binary_sensor.downstairs_occupancy_group",
				lightGroup: []string{
					"light.living_room",
					"light.office",
					"light.stairs_main_lights_2",
					"light.downstairs",
				},
			},
			{
				name:       "Bedroom",
				climate:    "climate.bedroom",
				occupancy:  "binary_sensor.bedroom_occupancy",
				lightGroup: []string{"light.bedroom_3"},
			},
		},
	}
}

func (c *Client) Name() string { return "homeassistant" }

type haState struct {
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

func (c *Client) getState(ctx context.Context, entityID string) (*haState, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/api/states/"+entityID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HA %s: status %d", entityID, resp.StatusCode)
	}
	var s haState
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("decode %s: %w", entityID, err)
	}
	return &s, nil
}

// Fetch queries all configured entities in parallel and folds them into
// "<presence>|<upstairs>|<downstairs>|<bedroom>", each area field already
// formatted as "AREA · temp° [· OCCUPIED] [· LIGHTS ON]". A failed
// individual lookup degrades that one piece rather than failing the
// whole scene — a single down entity shouldn't blank the whole card.
func (c *Client) Fetch(ctx context.Context) (string, error) {
	var wg sync.WaitGroup
	var presence string
	areaText := make([]string, len(c.areas))

	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := c.getState(ctx, c.presenceEntity)
		if err != nil || s == nil {
			presence = "?"
			return
		}
		if s.State == "on" {
			presence = "HOME"
		} else {
			presence = "AWAY"
		}
	}()

	for i, a := range c.areas {
		i, a := i, a
		wg.Add(1)
		go func() {
			defer wg.Done()
			areaText[i] = c.fetchArea(ctx, a)
		}()
	}

	wg.Wait()

	return presence + "|" + strings.Join(areaText, "|"), nil
}

// fetchArea queries one area's climate, occupancy, and light group
// concurrently and folds them into "AREA · temp° [· OCCUPIED] [· LIGHTS
// ON]".
func (c *Client) fetchArea(ctx context.Context, a area) string {
	var wg sync.WaitGroup
	temp := "—"
	var occupied, lightsOn bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := c.getState(ctx, a.climate)
		if err != nil || s == nil {
			return
		}
		if t, ok := s.Attributes["current_temperature"].(float64); ok {
			temp = strconv.Itoa(int(t)) + "°"
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := c.getState(ctx, a.occupancy)
		if err == nil && s != nil && s.State == "on" {
			occupied = true
		}
	}()

	for _, entityID := range a.lightGroup {
		entityID := entityID
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := c.getState(ctx, entityID)
			if err == nil && s != nil && s.State == "on" {
				lightsOn = true
			}
		}()
	}

	wg.Wait()

	text := strings.ToUpper(a.name) + " · " + temp
	if occupied {
		text += " · OCCUPIED"
	}
	if lightsOn {
		text += " · LIGHTS ON"
	}
	return text
}
