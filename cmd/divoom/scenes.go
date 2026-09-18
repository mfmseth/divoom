package main

import (
	"fmt"
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

// Font IDs on the device. Custom-pushed via adb (see docs/api.md →
// "Fonts on disk"). Iosevka for digits/mono rows, Roboto Condensed for
// prose.
const (
	fontMono  = 7 // Iosevka — numbers, mono rows
	fontProse = 9 // Roboto Condensed — labels, prose
)

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

// isoWeek returns the ISO 8601 week number for now (the second value of
// time.Time.ISOWeek).
func isoWeek(now time.Time) int {
	_, w := now.ISOWeek()
	return w
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

// weekendStatus returns the operator-footer right-hand string and its
// FontColor. Weekend window is Friday 18:00 through Monday 03:00 (local
// time) — inside it the text becomes "weekend!" in cYellow as a small
// festive marker. Outside it, the row reverts to the dim countdown
// "weekend+Nd" (Mon-Thu after 3am: 4..1 days; Fri before 6pm: +0d).
func weekendStatus(now time.Time) (text, color string) {
	wd := now.Weekday()
	hour := now.Hour()
	weekend := false
	switch wd {
	case time.Saturday, time.Sunday:
		weekend = true
	case time.Friday:
		weekend = hour >= 18
	case time.Monday:
		weekend = hour < 3
	}
	if weekend {
		return "weekend!", cYellow
	}
	// Outside the window — countdown to Saturday morning.
	n := 5 - int(wd)
	if n < 0 {
		n = 0 // defensive; Friday-pre-6pm falls here as +0d
	}
	return fmt.Sprintf("weekend-%dd", n), cFgDark
}

// alwaysOn builds the shared header every scene installs on top of its
// own Elements — day name, big clock, and the date/weekend footer row.
// Wired in via scene.Driver.AlwaysOn (see serve.go).
func alwaysOn(now time.Time) []frame.DispElement {
	weekendText, weekendColor := weekendStatus(now)
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
		// Left half of the footer row — date / day-of-year / iso-week,
		// dim mono left-aligned.
		{
			ID: idFooter, Type: "Text",
			StartX: 40, StartY: 400, Width: 720, Height: 44,
			Align:     0,
			FontSize:  28,
			FontID:    fontMono,
			FontColor: cFgDark,
			BgColor:   cBgHard,
			TextMessage: fmt.Sprintf("%s  doy:%d  w:%d",
				now.Format("2006-01-02"),
				now.YearDay(),
				isoWeek(now)),
		},
		// Right half of the footer row — weekend status, right-aligned
		// so it can carry its own colour (cYellow during the Fri 6pm →
		// Mon 3am window, cFgDark otherwise) without recolouring the
		// numeric metadata to its left.
		{
			ID: idWeekend, Type: "Text",
			StartX: 40, StartY: 400, Width: 720, Height: 44,
			Align:       1,
			FontSize:    28,
			FontID:      fontMono,
			FontColor:   weekendColor,
			BgColor:     cBgHard,
			TextMessage: weekendText,
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
