package main

import (
	"strings"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// "homeassistant" — a big weather readout and three big area rows
// (Upstairs / Downstairs / Bedroom), all sized like the always-on
// clock: this is a 10.1" display meant to be read from across a room,
// so every row gets the same "its own big thing" treatment rather than
// small dense text. Flush-left (not centered) and colored with a
// single reserved accent per the Modernist-pairing design review — see
// areaRow. The widget emits
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>" —
// presence is fetched but not displayed (dropped per request). Each
// area field is "AREA · temp°@FLAGS": the "@FLAGS" suffix is stripped
// before display and used only to color the row, never shown — that's
// what keeps the visible text down to ~14-17 characters, short enough
// to render at FontSize 65 without the device clipping it (it clips
// rather than wraps text that overflows its box width).
//
//   - Weather: "<CONDITION> · temp°" alone, FontSize 65, clock-orange,
//     no background fill — same plain style as the always-on clock;
//     this is the scene's own hero/identity color, not the activity
//     accent (see areaRow).
//   - Area rows: "AREA · temp°", FontSize 65, flush-left, orange only
//     when that area is occupied or has a light on (the one reserved
//     accent for activity), otherwise the plain neutral foreground —
//     one per area in the order Upstairs / Downstairs / Bedroom.
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
				Align: 0, FontSize: 65, FontID: fontMono,
				FontColor: cOrange, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 40, StartY: 610, Width: 720, Height: 90,
				Align: 0, FontSize: 65, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub3, Type: "Text",
				StartX: 40, StartY: 730, Width: 720, Height: 90,
				Align: 0, FontSize: 65, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 40, StartY: 850, Width: 720, Height: 90,
				Align: 0, FontSize: 65, FontID: fontMono,
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
// "@FLAGS" suffix before display, and colors the whole line: the
// accent orange is reserved for activity, so it fires only when FLAGS
// is non-empty (that area is occupied or has a light on); otherwise
// the row stays the plain neutral foreground. Earlier revisions also
// banded the neutral case by temperature (cold/comfortable/warm/hot) —
// dropped per the design review's "one reserved accent" note, which
// reads a rainbow of non-accent colors as diluting what "colored"
// means on this scene.
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
		return text, cFg
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
