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

// The alert banner sits on-screen, showing the matching image, only
// when the widget reports Home Assistant or the internet down.
func TestAlertBannerShowsOnlyWhenDown(t *testing.T) {
	banner := func(status string) (x int, url string) {
		raw := "CLOUDY 61°||HOME|UPSTAIRS 76°@|DOWNSTAIRS 77°@|BEDROOM 76°@|" + status
		elements := append([]frame.DispElement(nil), homeAssistantScene(nil).Elements...)
		positionDynamicMarks(time.Now(), raw, elements)
		for _, e := range elements {
			if e.ID == idSceneBanner {
				return e.StartX, e.Url
			}
		}
		t.Fatal("no banner element")
		return 0, ""
	}
	if x, _ := banner(""); x+CanvasW > 0 {
		t.Errorf("all up: banner StartX = %d, want off-canvas", x)
	}
	for _, status := range []string{homeassistant.StatusDown, homeassistant.StatusISPDown} {
		x, url := banner(status)
		if x != 0 || url != haBanners[status].path {
			t.Errorf("%s: banner at x=%d url=%q, want x=0 url=%q", status, x, url, haBanners[status].path)
		}
	}
}

// Every phrase the HA helper can produce gets its own pushed image.
func TestWearPathsAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for _, phrase := range wearPhrases {
		path := wearPath(phrase)
		if prev, dup := seen[path]; dup {
			t.Errorf("%q and %q both map to %s", prev, phrase, path)
		}
		seen[path] = phrase
	}
	if got := wearPath("LONG SLEEVES + UMBRELLA"); got != "/userdata/wallclock_wear_long_sleeves_umbrella.png" {
		t.Errorf("wearPath = %q", got)
	}
}

// The wear line shows a known phrase's image and hides anything else.
func TestWearRowShowsOnlyKnownPhrases(t *testing.T) {
	wearRow := func(wear string) (x int, url string) {
		raw := "CLOUDY 61°||HOME|UPSTAIRS 76°@|DOWNSTAIRS 77°@|BEDROOM 76°@||" + wear
		elements := append([]frame.DispElement(nil), homeAssistantScene(nil).Elements...)
		positionDynamicMarks(time.Now(), raw, elements)
		for _, e := range elements {
			if e.ID == idSceneWear {
				return e.StartX, e.Url
			}
		}
		t.Fatal("no wear element")
		return 0, ""
	}
	if x, url := wearRow("LONG SLEEVES"); x != 0 || url != wearPath("LONG SLEEVES") {
		t.Errorf("known phrase: x=%d url=%q", x, url)
	}
	for _, wear := range []string{"", "unavailable", "PARKA"} {
		if x, _ := wearRow(wear); x+CanvasW > 0 {
			t.Errorf("%q: wear row at x=%d, want off-canvas", wear, x)
		}
	}
}
