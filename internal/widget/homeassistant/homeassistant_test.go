package homeassistant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// status is the last pipe field of Fetch's output.
func fetchStatus(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	defer srv.Close()
	raw, err := New(srv.URL, "token").Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	parts := strings.Split(raw, "|")
	if len(parts) != 8 {
		t.Fatalf("Fetch = %q, want 8 pipe fields", raw)
	}
	return parts[6]
}

func TestFetchReportsDownWhenNothingAnswers(t *testing.T) {
	got := fetchStatus(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	if got != StatusDown {
		t.Errorf("status = %q, want %q", got, StatusDown)
	}
}

func TestFetchNotDownWhenSomeEntitiesAnswer(t *testing.T) {
	got := fetchStatus(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/states/climate.upstairs":
			w.Write([]byte(`{"state":"off","attributes":{"current_temperature":76}}`))
		case "/api/states/sensor.ucg_fiber_google_wan2_latency":
			w.Write([]byte(`{"state":"12","attributes":{}}`))
		default:
			http.NotFound(w, r)
		}
	})
	if got != "" {
		t.Errorf("status = %q, want empty (HA answers, one WAN probe has a reading)", got)
	}
}

func TestFetchReportsISPDownWhenNoWANReading(t *testing.T) {
	got := fetchStatus(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "_latency") {
			w.Write([]byte(`{"state":"unknown","attributes":{}}`))
			return
		}
		w.Write([]byte(`{"state":"off","attributes":{"current_temperature":76}}`))
	})
	if got != StatusISPDown {
		t.Errorf("status = %q, want %q", got, StatusISPDown)
	}
}
