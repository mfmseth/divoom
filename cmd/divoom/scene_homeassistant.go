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

// "homeassistant" — presence plus one row per area (Upstairs /
// Downstairs / Bedroom), each row already folding that area's climate,
// occupancy, and lights-on state into a single self-labeled line — no
// separate section headers or divider rules needed. The widget emits
// "<presence>|<upstairs>|<downstairs>|<bedroom>", with each area field
// pre-formatted as "AREA · temp° [· OCCUPIED] [· LIGHTS ON]".
//
//   - Presence: HOME or AWAY, filled in an accent-orange chip when HOME.
//   - Area rows: plain text, one per area, in the order Upstairs /
//     Downstairs / Bedroom.
func homeAssistantScene(widgets map[string]widget.Widget) *scene.Scene {
	return &scene.Scene{
		Name:   "homeassistant",
		Weight: WeightInformational,
		BgPath: bgHomeAssistant,
		Elements: []frame.DispElement{
			{
				ID: idSceneMain, Type: "Text",
				StartX: 80, StartY: 520, Width: 640, Height: 190,
				Align: 2, FontSize: 150, FontID: fontProse,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub1, Type: "Text",
				StartX: 80, StartY: 790, Width: 640, Height: 70,
				Align: 0, FontSize: 38, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneSub4, Type: "Text",
				StartX: 80, StartY: 930, Width: 640, Height: 70,
				Align: 0, FontSize: 38, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
			{
				ID: idSceneTitle, Type: "Text",
				StartX: 80, StartY: 1070, Width: 640, Height: 70,
				Align: 0, FontSize: 38, FontID: fontMono,
				FontColor: cFg, BgColor: cBgHard,
			},
		},
		Widget: widgets["homeassistant"],
		Mounts: []scene.Mount{
			{ID: idSceneMain, Format: haPresence},
			{ID: idSceneSub1, Format: pipeAt(1)},
			{ID: idSceneSub4, Format: pipeAt(2)},
			{ID: idSceneTitle, Format: pipeAt(3)},
		},
		OnActivate: haChipColorize,
	}
}

// idSceneSub4 is a fourth "sub" Text slot, alongside the shared
// idSceneSub1-3 pool in scenes.go.
const idSceneSub4 = 14

// bgHomeAssistant is the on-device path for this scene's background —
// the only one that exists now that every other scene has been removed.
const bgHomeAssistant = "/userdata/wallclock_bg_homeassistant.jpg"

func haPresence(raw string) (text, color string) {
	return strings.ToUpper(weatherPipeField(raw, 0)), cFg
}

// haChipColorize fills idSceneMain's BgColor with the accent when
// presence is HOME, and swaps FontColor to the dark bg color for
// contrast against that fill. AWAY keeps the default text-on-hero-bg
// look set in the Elements above.
func haChipColorize(_ time.Time, raw string, elements []frame.DispElement) {
	presence := weatherPipeField(raw, 0)
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
