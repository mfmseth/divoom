// Package homeassistant queries a Home Assistant instance's REST API,
// grouped by area (Upstairs / Downstairs / Bedroom), and emits a
// pipe-separated
// "<weather text>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>"
// string for the homeassistant scene — each area field already combines
// that area's climate, occupancy, and lights-on state into one line, and
// icon is "rain", "snow", or "" depending on today's forecast.
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
	weatherEntity  string
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
		weatherEntity:  "weather.forecast_home",
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
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>", each
// area field already formatted as "AREA · temp° [· OCCUPIED] [· LIGHTS
// ON]" — the worst case (Downstairs with both flags) is 40 characters,
// which the scene's area-row font size is sized to fit; the device
// clips (rather than wraps) text that overflows its box width, so
// don't grow this without also checking that row's FontSize/Width in
// scene_homeassistant.go. A failed individual lookup degrades that one
// piece rather than failing the whole scene — a single down entity
// shouldn't blank the whole card.
func (c *Client) Fetch(ctx context.Context) (string, error) {
	var wg sync.WaitGroup
	var presence, weatherText, icon string
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

	wg.Add(1)
	go func() {
		defer wg.Done()
		weatherText, icon = c.fetchWeather(ctx)
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

	return weatherText + "|" + icon + "|" + presence + "|" + strings.Join(areaText, "|"), nil
}

// forecastDay is the one field we need out of weather.get_forecasts'
// daily response.
type forecastDay struct {
	Condition string `json:"condition"`
}

// fetchWeather returns "<CONDITION> <temp>°" from the weather entity's
// current state, and an icon hint ("rain", "snow", or "") derived from
// today's daily forecast condition — not the current condition, since a
// clear-now-rain-later day should still show the icon.
func (c *Client) fetchWeather(ctx context.Context) (text, icon string) {
	s, err := c.getState(ctx, c.weatherEntity)
	if err != nil || s == nil {
		return "WEATHER · —", ""
	}
	temp := "—"
	if t, ok := s.Attributes["temperature"].(float64); ok {
		temp = strconv.Itoa(int(t)) + "°"
	}
	text = strings.ToUpper(s.State) + " · " + temp

	today, err := c.fetchTodayForecast(ctx)
	if err != nil || today == "" {
		return text, ""
	}
	return text, iconFor(today)
}

// fetchTodayForecast calls the weather.get_forecasts service (REST
// service-call endpoint, ?return_response so the forecast comes back in
// the response body) and returns the first (today's) daily entry's
// condition string.
func (c *Client) fetchTodayForecast(ctx context.Context) (string, error) {
	body := fmt.Sprintf(`{"entity_id":%q,"type":"daily"}`, c.weatherEntity)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/services/weather/get_forecasts?return_response",
		strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HA get_forecasts: status %d", resp.StatusCode)
	}
	var out struct {
		ServiceResponse map[string]struct {
			Forecast []forecastDay `json:"forecast"`
		} `json:"service_response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode get_forecasts: %w", err)
	}
	fc, ok := out.ServiceResponse[c.weatherEntity]
	if !ok || len(fc.Forecast) == 0 {
		return "", nil
	}
	return fc.Forecast[0].Condition, nil
}

// iconFor maps an HA weather condition string to our icon hint. HA's
// condition set includes "rainy", "pouring", "lightning-rainy",
// "snowy", and "snowy-rainy" among others (see the weather integration
// docs) — anything containing "rain"/"pour" gets the rain icon, anything
// containing "snow" gets the snow icon (checked first since
// "snowy-rainy" should read as snow, not rain).
func iconFor(condition string) string {
	c := strings.ToLower(condition)
	switch {
	case strings.Contains(c, "snow"):
		return "snow"
	case strings.Contains(c, "rain"), strings.Contains(c, "pour"):
		return "rain"
	default:
		return ""
	}
}

// fetchArea queries one area's climate, occupancy, and light group
// concurrently and folds them into "AREA · temp° [· OCCUPIED] [·
// LIGHTS ON]".
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
