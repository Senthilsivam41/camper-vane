package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,https://app.example.com")

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	h := CORSMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Allow-Origin=%q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials=%q", got)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestCORSMiddlewarePreflight(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")

	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler should not run on OPTIONS")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/chat/stream", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Allow-Methods on preflight")
	}
}

func TestCORSMiddlewarePassThroughWhenUnset(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected no CORS headers when unset")
	}
	if w.Code != http.StatusTeapot {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestCORSMiddlewareRejectsDisallowedOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")

	h := CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Origin", "https://evil.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected no CORS headers for disallowed origin")
	}
}

func TestParseCORSOriginsRejectsWildcard(t *testing.T) {
	cases := []string{"*", "https://app.example.com,*", "*,https://app.example.com"}
	for _, raw := range cases {
		_, err := parseCORSOrigins(raw)
		if err == nil {
			t.Fatalf("expected error for %q", raw)
		}
		if !strings.Contains(err.Error(), "*") {
			t.Fatalf("error for %q should mention wildcard: %v", raw, err)
		}
	}
}

func TestParseCORSOriginsExactOnly(t *testing.T) {
	got, err := parseCORSOrigins("http://localhost:5173, https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "http://localhost:5173" || got[1] != "https://app.example.com" {
		t.Fatalf("got=%v", got)
	}
	empty, err := parseCORSOrigins("")
	if err != nil || empty != nil {
		t.Fatalf("empty=%v err=%v", empty, err)
	}
}
