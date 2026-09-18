package main

import (
	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "homeassistant" — a big weather readout sized like the always-on
// clock (its own thing, not sharing a line with anything else), and
// one row per area (Upstairs / Downstairs / Bedroom), each area row
// already folding that area's climate, occupancy, and lights-on state
// into a single self-labeled line. The widget emits
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>" —
// presence is fetched but not displayed (dropped per request; only
// the area rows' OCC/LIT flags carry activity info now) — with each
// area field pre-formatted as "AREA · temp° [· OCC] [· LIT]", kept
// short since the device clips (rather than wraps) text that overflows
// its box width.
//
//   - Weather: "<CONDITION> · temp°" alone, FontSize 65, clock-orange,
//     no background fill — same plain style as the always-on clock,
//     just its own row instead of sharing one with presence.
//   - Area rows: plain text, one per area, in the order Upstairs /
//     Downstairs / Bedroom.
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
				StartX: 60, StartY: 650, Width: 680, Height: 60,
				Align: 0, FontSize: 34, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 60, StartY: 725, Width: 680, Height: 60,
				Align: 0, FontSize: 34, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 60, StartY: 800, Width: 680, Height: 60,
				Align: 0, FontSize: 34, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["homeassistant"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: pipeAt(0)},
			{ID: idSceneSub1, Format: pipeAt(3)},
			{ID: idSceneSub3, Format: pipeAt(4)},
			{ID: idSceneTitle, Format: pipeAt(5)},
		},
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
