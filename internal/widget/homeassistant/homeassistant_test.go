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
	if len(parts) != 7 {
		t.Fatalf("Fetch = %q, want 7 pipe fields", raw)
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
		if r.URL.Path == "/api/states/climate.upstairs" {
			w.Write([]byte(`{"state":"off","attributes":{"current_temperature":76}}`))
			return
		}
		http.NotFound(w, r)
	})
	if got != "" {
		t.Errorf("status = %q, want empty (only some entities missing)", got)
	}
}
