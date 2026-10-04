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

// Element IDs. Always-on top reserves 1-4; scene primaries start at 9;
// the dynamically-positioned Image elements (see scene_homeassistant.go)
// start at 19 to stay clearly clear of the Text-cap range. Each scene's
// layout is its own install, so re-using IDs across scenes is fine; we
// keep the IDs distinct only within a single scene.
const (
	idHeader = 1
	idTime   = 2

	idSceneWeather     = 9
	idSceneRoomBase    = 10 // +0..2 for Upstairs/Downstairs/Bedroom
	idSceneWeatherIcon = 19
	idSceneMarkBase    = 20 // +0..2, one Image per room's occupancy mark
	idSceneBanner      = 23
)

// WeightInformational is the base weight for the homeassistant scene in
// the driver's weighted-random pick — moot with a single scene (it just
// needs to be > 0 so the driver doesn't skip it), kept as a named
// constant for clarity over a bare literal.
const WeightInformational = 40

// Device font IDs for the two Archivo weights the wallclock-scene design
// review calls for (custom-pushed via adb; see docs/api.md "Fonts on
// disk" and scripts/download-fonts.sh). SemiBold carries the header,
// weather, and room rows; ExtraBold is reserved for the clock alone.
const (
	fontArchivoSemiBold  = 7
	fontArchivoExtraBold = 8
)

// archivoSemiBoldFile is the local TTF basename `render.MeasureLabel`
// loads to measure room-row text width when placing an occupancy mark
// (see scene_homeassistant.go) — must stay in sync with fontArchivoSemiBold.
const archivoSemiBoldFile = "Archivo-SemiBold.ttf"

// Design tokens. cHaAccent500 is reserved for occupancy marks only.
// cHaTextAccent replaces the design review's original neutral-400 gray
// (header/weather/dim text) per user feedback wanting a brighter "pop
// of color" there instead of a flat gray — accent-400 from the same
// design system ramp, distinct enough from cHaAccent500 that an
// occupancy mark doesn't blend into it.
const (
	cHaNeutral100 = "#f8f4f4"
	cHaNeutral700 = "#605d5d"
	cHaNeutral900 = "#2d2b2b"
	cHaAccent500  = "#ff563c"
	cHaTextAccent = "#ff9783"
)

// alwaysOn builds the shared header every scene installs on top of its
// own Elements — one combined weekday+date row, and the big clock.
// Wired in via scene.Driver.AlwaysOn (see serve.go). Letter-spacing from
// the design review (0.02-0.04em) has no device-API equivalent and is
// skipped.
//
// The clock is a custom Text element, not the device's built-in Time
// type: the built-in clock self-updates but a live test against the
// frame confirmed its 12-hour mode shows no AM/PM indicator at all
// (device firmware limitation) -- there's no way to get AM/PM out of
// it. Rendering the time ourselves means the daemon has to keep it
// current: tickClock (serve.go) patches the clock and header text at
// every minute boundary between scene reinstalls. FontSize dropped from the bare-digits 169 to 140
// -- "12:32 PM" (the widest case) measures ~651px at 140 in Archivo
// ExtraBold, fitting the 700px box; 169 would have clipped it.
func alwaysOn(now time.Time) []frame.DispElement {
	return []frame.DispElement{
		{
			ID: idHeader, Type: "Text",
			StartX: 40, StartY: 30, Width: 720, Height: 80,
			Align:       2,
			FontSize:    56,
			FontID:      fontArchivoSemiBold,
			FontColor:   cHaTextAccent,
			BgColor:     cHaNeutral900,
			TextMessage: headerText(now),
		},
		{
			ID: idTime, Type: "Text",
			StartX: 50, StartY: 165, Width: 700, Height: 200,
			Align:       2,
			FontSize:    140,
			FontID:      fontArchivoExtraBold,
			FontColor:   cHaNeutral100,
			BgColor:     cHaNeutral900,
			TextMessage: clockText(now),
		},
	}
}

// alwaysOnTexts is the in-place text patch that brings alwaysOn's Text
// elements up to date for now, without reinstalling the layout.
func alwaysOnTexts(now time.Time) []frame.TextUpdate {
	return []frame.TextUpdate{
		{ID: idHeader, TextMessage: headerText(now)},
		{ID: idTime, TextMessage: clockText(now)},
	}
}

func headerText(now time.Time) string {
	return strings.ToUpper(now.Weekday().String()) + " · " + now.Format("01-02-2006")
}

func clockText(now time.Time) string {
	return now.Format("3:04 PM")
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
