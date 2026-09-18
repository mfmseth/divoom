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

// "homeassistant" — a top weather row, a compact presence chip, and one
// row per area (Upstairs / Downstairs / Bedroom), each area row already
// folding that area's climate, occupancy, and lights-on state into a
// single self-labeled line. The widget emits
// "<weather>|<icon>|<presence>|<upstairs>|<downstairs>|<bedroom>", with
// each area field pre-formatted as "AREA · temp° [· OCC] [· LIT]" — kept
// short since the device clips (rather than wraps) text that overflows
// its box width.
//
//   - Weather: "<CONDITION> · temp°", with a rain/snow cloud icon baked
//     into the bg (via BgPathFor) when today's forecast calls for it.
//   - Presence: HOME or AWAY, filled in an accent-orange chip when HOME.
//     Deliberately small (FontSize 70, not a giant hero number) so it
//     reads as one line among the others instead of a big gap-creating
//     block between weather and the area rows.
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
				ID: idSceneSub2, Type: "Text",
				StartX: 80, StartY: 520, Width: 500, Height: 50,
				Align: 0, FontSize: 40, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 580, Width: 640, Height: 90,
				Align: 2, FontSize: 70, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 60, StartY: 700, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 60, StartY: 765, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 60, StartY: 830, Width: 680, Height: 55,
				Align: 0, FontSize: 30, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["homeassistant"],
		Mounts: []scene.Mount{
			{ID: idSceneSub2, Format: pipeAt(0)},
			{ID: idSceneMain, Format: haPresence},
			{ID: idSceneSub1, Format: pipeAt(3)},
			{ID: idSceneSub4, Format: pipeAt(4)},
			{ID: idSceneTitle, Format: pipeAt(5)},
		},
		OnActivate: haChipColorize,
	}
}

// idSceneSub4 is a fourth "sub" Text slot, alongside the shared
// idSceneSub1-3 pool in scenes.go.
const idSceneSub4 = 14

// On-device bg paths for this scene. Three variants -- plain, rain-icon,
// snow-icon -- all pre-pushed at startup; the scene's BgPathFor picks
// among them per activation based on the widget's icon field.
const (
	bgHomeAssistant     = "/userdata/wallclock_bg_homeassistant.jpg"
	bgHomeAssistantRain = "/userdata/wallclock_bg_homeassistant_rain.jpg"
	bgHomeAssistantSnow = "/userdata/wallclock_bg_homeassistant_snow.jpg"
)

func haPresence(raw string) (text, color string) {
	return strings.ToUpper(weatherPipeField(raw, 2)), cFg
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
