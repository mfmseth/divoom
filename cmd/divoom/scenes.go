package main

import (
	"strings"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/render"
	"github.com/dragonpaw/divoom/internal/scene"
	"github.com/dragonpaw/divoom/internal/widget"
)

// CanvasW shadows render.CanvasW so scene-layout math reads naturally.
const CanvasW = render.CanvasW

// Element IDs. Always-on top reserves 1-4; scene primaries start at 9.
// Each scene's layout is its own install, so re-using IDs across scenes is
// fine; we keep the IDs distinct only within a single scene.
const (
	idDay     = 1
	idTime    = 2
	idFooter  = 3
	idWeekend = 4

	idSceneTitle = 9
	idSceneMain  = 10
	idSceneSub1  = 11
	idSceneSub2  = 12
	idSceneSub3  = 13
)

// WeightInformational is the base weight for the homeassistant scene in
// the driver's weighted-random pick — moot with a single scene (it just
// needs to be > 0 so the driver doesn't skip it), kept as a named
// constant for clarity over a bare literal.
const WeightInformational = 40

// fontMono is the device font ID for Archivo Black, this scene's only
// non-stock typography -- single family per the Modernist-pairing
// design review, used at multiple sizes rather than multiple weights
// (custom-pushed via adb; see docs/api.md "Fonts on disk" and
// scripts/download-fonts.sh for why it's a standalone static weight
// rather than an instance of the variable Archivo[wdth,wght] font).
const fontMono = 7

// Gruvbox semantic colors. Reds and greens signal direction (down/up);
// yellow / blue / aqua signal weather conditions; fg / fg-dark are quiet.
const (
	cFg     = "#ebdbb2"
	cFgDark = "#a89984"
	cRed    = "#fb4934"
	cGreen  = "#b8bb26"
	cYellow = "#fabd2f"
	cBlue   = "#83a598"
	cAqua   = "#8ec07b"
	cPurple = "#d3869b"
	cOrange = "#fe8019"
	cBgHard = "#1d2021"
)

// dayColors picks a gruvbox accent per weekday, sweeping the palette
// through the week so each day reads distinctly at a glance.
var dayColors = map[time.Weekday]string{
	time.Sunday:    cPurple,
	time.Monday:    cRed,
	time.Tuesday:   cOrange,
	time.Wednesday: cYellow,
	time.Thursday:  cGreen,
	time.Friday:    cAqua,
	time.Saturday:  cBlue,
}

// timeColor returns the AM/PM accent for the always-on clock — cAqua
// mornings, cOrange afternoons/evenings — so the clock reads warm or
// cool at a glance.
func timeColor(now time.Time) string {
	if now.Hour() < 12 {
		return cAqua
	}
	return cOrange
}

// alwaysOn builds the shared header every scene installs on top of its
// own Elements — day name, big clock, and the date footer row. The
// footer used to have a second (weekend-countdown) half on its right,
// dropped to free a Text slot: the homeassistant scene needs weather
// and presence as two separate elements (each "its own thing," like
// the clock), which pushes the scene to 5 Text elements -- 1 header +
// 5 scene is the most this device's 6-Text cap allows.
// Wired in via scene.Driver.AlwaysOn (see serve.go).
func alwaysOn(now time.Time) []frame.DispElement {
	return []frame.DispElement{
		{
			// Week is a device built-in (renders the day name from
			// the device's own clock). Doesn't count against the
			// 6-Text cap. The "> " prompt to its left is baked into
			// every scene bg by buildHeroImage; this element only
			// owns the day name itself. StartX shifted right of the
			// baked prompt; FontColor still picks up the per-day
			// chroma so each weekday has its own colour.
			ID: idDay, Type: "Week",
			StartX: 110, StartY: 30, Width: 650, Height: 80,
			Align:     0,
			FontSize:  64,
			FontID:    fontMono,
			FontColor: dayColors[now.Weekday()],
			BgColor:   cBgHard,
		},
		{
			ID: idTime, Type: "Time",
			StartX: 50, StartY: 140, Width: 700, Height: 200,
			Align:     2,
			FontSize:  160,
			FontID:    fontMono,
			FontColor: timeColor(now),
			BgColor:   cBgHard,
		},
		// Date, right-aligned on the same row as the weekday (which
		// sits on the left via the baked "> " prompt + idDay below) --
		// opposite sides of one header row instead of a separate
		// footer line. MM-DD-YYYY per request.
		{
			ID: idFooter, Type: "Text",
			StartX: 40, StartY: 30, Width: 720, Height: 80,
			Align:       1,
			FontSize:    64,
			FontID:      fontMono,
			FontColor:   cFgDark,
			BgColor:     cBgHard,
			TextMessage: now.Format("01-02-2006"),
		},
	}
}

// pipeAt returns a Mount.Format closure that picks segment i of a
// pipe-separated raw widget string, with no color override.
func pipeAt(i int) func(raw string) (text, color string) {
	return func(raw string) (text, color string) {
		parts := strings.Split(raw, "|")
		if i < 0 || i >= len(parts) {
			return "", ""
		}
		return parts[i], ""
	}
}

// weatherPipeField pulls segment i of a pipe-separated raw string,
// returning "" if the string has fewer than i+1 segments. Named for its
// original use in the weather scene; reused by scene_homeassistant.go.
func weatherPipeField(raw string, i int) string {
	parts := strings.Split(raw, "|")
	if i >= len(parts) {
		return ""
	}
	return parts[i]
}

// buildScenes returns the configured scene rotation — just
// "homeassistant" in this fork; every other scene from upstream has
// been removed. `widgets` maps a scene's Name to the Widget that
// supplies its dynamic text.
func buildScenes(widgets map[string]widget.Widget) []*scene.Scene {
	return []*scene.Scene{homeAssistantScene(widgets)}
}
