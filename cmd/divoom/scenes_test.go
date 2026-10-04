package main

import (
	"testing"
	"time"

	"github.com/dragonpaw/divoom/internal/frame"
	"github.com/dragonpaw/divoom/internal/widget/homeassistant"
)

// The minute-by-minute clock patch must hit the same element IDs, with
// the same text, that a full reinstall would set — otherwise the clock
// or date would visibly jump on every scene reinstall.
func TestAlwaysOnTextsMatchesAlwaysOn(t *testing.T) {
	now := time.Date(2026, time.October, 4, 13, 7, 0, 0, time.Local)
	installed := map[int]string{}
	for _, e := range alwaysOn(now) {
		if e.Type == "Text" {
			installed[e.ID] = e.TextMessage
		}
	}
	updates := alwaysOnTexts(now)
	if len(updates) != len(installed) {
		t.Fatalf("got %d updates, want one per installed Text element (%d)", len(updates), len(installed))
	}
	for _, u := range updates {
		if want, ok := installed[u.ID]; !ok || u.TextMessage != want {
			t.Errorf("update ID %d = %q, installed text = %q (present: %v)", u.ID, u.TextMessage, want, ok)
		}
	}
	if got := clockText(now); got != "1:07 PM" {
		t.Errorf("clockText = %q, want %q", got, "1:07 PM")
	}
}

// The HA-down banner sits on-screen only when the widget reports Home
// Assistant down.
func TestHABannerShowsOnlyWhenDown(t *testing.T) {
	bannerX := func(raw string) int {
		s := homeAssistantScene(nil)
		elements := append([]frame.DispElement(nil), s.Elements...)
		positionDynamicMarks(time.Now(), raw, elements)
		for _, e := range elements {
			if e.ID == idSceneHABanner {
				return e.StartX
			}
		}
		t.Fatal("no banner element")
		return 0
	}
	up := "CLOUDY 61°||HOME|UPSTAIRS 76°@|DOWNSTAIRS 77°@|BEDROOM 76°@|"
	down := "WEATHER —||?|UPSTAIRS —@|DOWNSTAIRS —@|BEDROOM —@|" + homeassistant.StatusDown
	if x := bannerX(up); x+CanvasW > 0 {
		t.Errorf("HA up: banner StartX = %d, want off-canvas", x)
	}
	if x := bannerX(down); x != 0 {
		t.Errorf("HA down: banner StartX = %d, want 0", x)
	}
}
