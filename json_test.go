package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Golbat now returns a JSON error body alongside the 404 status code, e.g.
// {"title":"Not Found","status":404,"detail":"pokemon not found"}
// getJson must not decode that error body into target, otherwise callers
// see a non-nil "record" and treat the error as a valid result.
func TestGetJson_NotFoundWithBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"pokemon not found"}`))
	}))
	defer srv.Close()

	var target any
	if err := getJson(srv.URL, &target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target != nil {
		t.Fatalf("expected target to stay nil on 404, got %#v", target)
	}
}

// Old golbat behaviour: 404 with an empty body must also leave target nil.
func TestGetJson_NotFoundEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var target any
	_ = getJson(srv.URL, &target)
	if target != nil {
		t.Fatalf("expected target to stay nil on empty 404, got %#v", target)
	}
}

// A successful response must be decoded into target as before.
func TestGetJson_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"lat":1.5,"lon":2.5}`))
	}))
	defer srv.Close()

	var target any
	if err := getJson(srv.URL, &target); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := target.(map[string]any)
	if !ok {
		t.Fatalf("expected decoded map, got %#v", target)
	}
	if m["lat"] != 1.5 {
		t.Fatalf("expected lat 1.5, got %#v", m["lat"])
	}
}
