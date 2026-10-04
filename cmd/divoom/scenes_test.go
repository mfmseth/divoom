package main

import (
	"testing"
	"time"
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
