package main

import (
	"strconv"
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "homeassistant" — a big weather readout and three big area rows
// (Upstairs / Downstairs / Bedroom), all sized like the always-on
// clock: this is a 10.1" display meant to be read from across a room,
// so every row gets the same "its own big thing" treatment rather than
// small dense text. The widget emits
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>" —
// presence is fetched but not displayed (dropped per request). Each
// area field is "AREA · temp°@FLAGS": the "@FLAGS" suffix is stripped
// before display and used only to color the row (see areaRow) — that's
// what keeps the visible text down to ~14-17 characters, short enough
// to render at FontSize 65 without the device clipping it (it clips
// rather than wraps text that overflows its box width).
//
//   - Weather: "<CONDITION> · temp°" alone, FontSize 65, clock-orange,
//     no background fill — same plain style as the always-on clock.
//   - Area rows: "AREA · temp°", FontSize 65, centered, colored by
//     activity/temperature (see areaRow) — same scale and treatment as
//     the weather row, one per area in the order Upstairs / Downstairs
//     / Bedroom.
func homeAssistantScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "homeassistant",
		Weight: WeightInformational,
		BgPath: bgHomeAssistant,
		BgPathFor: func(raw string) string {
			switch weatherPipeField(raw, 1) {
			case "rain":
				return bgHomeAssistantRain
			case "snow":
				return bgHomeAssistantSnow
			default:
				return bgHomeAssistant
			}
		},
		Elements: []frame.DispElement{
			{
				ID: idSceneMain, Type: "Text",
				StartX: 40, StartY: 480, Width: 720, Height: 100,
				Align: 2, FontSize: 65, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 610, Width: 720, Height: 90,
				Align: 2, FontSize: 65, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 40, StartY: 730, Width: 720, Height: 90,
				Align: 2, FontSize: 65, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 40, StartY: 850, Width: 720, Height: 90,
				Align: 2, FontSize: 65, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["homeassistant"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: areaRow(3)},
			{ID: idSceneSub3, Format: areaRow(4)},
			{ID: idSceneTitle, Format: areaRow(5)},
		},
	}
}

// areaRow returns a Mount.Format closure that picks segment i (one of
// the widget's "AREA · temp°@FLAGS" strings), strips the hidden
// "@FLAGS" suffix before display, and colors the whole line so
// activity/temperature reads at a glance without the row needing to
// spell out OCCUPIED/LIGHTS ON in text:
//
//   - FLAGS non-empty (area occupied or has a light on) → accent
//     orange, so any activity jumps out regardless of temperature.
//   - Otherwise → banded by temperature, same comfort logic the old
//     weather scene used: cold blue-ish (cAqua), comfortable green,
//     warm yellow, hot red.
//
// A single Text element only carries one color for its whole string —
// there's no way to tint just the temperature differently within one
// line — so this picks the single most useful signal per row instead.
func areaRow(i int) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		field := weatherPipeField(raw, i)
		if field == "" {
			return "", ""
		}
		text, flags, _ := strings.Cut(field, "@")
		if flags != "" {
			return text, cOrange
		}
		return text, areaTempColor(text)
	}
}

// areaTempColor pulls the integer temperature out of an "AREA · temp°
// ..." string (the digits immediately before the first "°") and bands
// it into a comfort color. Defensive: any parse failure (missing °,
// non-numeric prefix) falls back to the quiet default cFg rather than
// guessing.
func areaTempColor(text string) string {
	idx := strings.IndexRune(text, '°')
	if idx < 0 {
		return cFg
	}
	start := idx
	for start > 0 && text[start-1] >= '0' && text[start-1] <= '9' {
		start--
	}
	if start == idx {
		return cFg
	}
	n, err := strconv.Atoi(text[start:idx])
	if err != nil {
		return cFg
	}
	switch {
	case n < 60:
		return cAqua
	case n <= 75:
		return cGreen
	case n <= 82:
		return cYellow
	default:
		return cRed
	}
}

// On-device bg paths for this scene. Three variants -- plain, rain-icon,
// snow-icon -- all pre-pushed at startup; the scene's BgPathFor picks
// among them per activation based on the widget's icon field.
const (
	bgHomeAssistant     = "/userdata/wallclock_bg_homeassistant.jpg"
	bgHomeAssistantRain = "/userdata/wallclock_bg_homeassistant_rain.jpg"
	bgHomeAssistantSnow = "/userdata/wallclock_bg_homeassistant_snow.jpg"
)
