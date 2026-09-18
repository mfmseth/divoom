package main

import (
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// mhAccent reuses the same orange the always-on clock (idTime) renders
// in during the afternoon/evening (see timeColor/cOrange in scenes.go),
// so the presence chip reads as part of the same visual system instead
// of introducing a second accent color.
const mhAccent = cOrange

// "homeassistant" — a combined weather+presence row, and one row per
// area (Upstairs / Downstairs / Bedroom), each area row already folding
// that area's climate, occupancy, and lights-on state into a single
// self-labeled line. The widget emits
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>", with
// each area field pre-formatted as "AREA · temp° [· OCC] [· LIT]" — kept
// short since the device clips (rather than wraps) text that overflows
// its box width.
//
// Weather and presence share ONE Text element (not two) on purpose: the
// device's AlwaysOn header already spends 2 of its 6-Text budget on the
// date/weekend footer, so 4 scene rows is the most this scene can use —
// a 5th (weather and presence as separate rows) silently dropped
// whichever Text element landed last in the install (Bedroom), since
// the always-on elements are prepended ahead of the scene's own.
//
//   - Weather + presence: "<CONDITION> · temp° · HOME", filled in an
//     accent-orange chip when presence is HOME.
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
				StartX: 60, StartY: 540, Width: 680, Height: 60,
				Align: 2, FontSize: 40, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 60, StartY: 680, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 60, StartY: 745, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 60, StartY: 810, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["homeassistant"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: haWeatherAndPresence},
			{ID: idSceneSub1, Format: pipeAt(3)},
			{ID: idSceneSub3, Format: pipeAt(4)},
			{ID: idSceneTitle, Format: pipeAt(5)},
		},
		OnActivate: haChipColorize,
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

func haWeatherAndPresence(raw string) (text, color string) {
	weather := weatherPipeField(raw, 0)
	presence := strings.ToUpper(weatherPipeField(raw, 2))
	return weather + " · " + presence, cOrange
}

// haChipColorize fills idSceneMain's BgColor with the accent when
// presence is HOME, and swaps FontColor to the dark bg color for
// contrast against that fill. AWAY keeps the default text-on-hero-bg
// look set in the Elements above.
func haChipColorize(_ time.Time, raw string, elements []frame.DispElement) {
	presence := weatherPipeField(raw, 2)
	if presence != "HOME" {
		return
	}
	for i := range elements {
		if elements[i].ID == idSceneMain {
			elements[i].BgColor = mhAccent
			elements[i].FontColor = cBgHard
		}
	}
}
