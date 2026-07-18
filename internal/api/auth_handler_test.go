package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"camper-vane/internal/auth"
	"camper-vane/internal/db"
)

func TestAuthHandler(t *testing.T) {
	repo, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	handler := NewAuthHandler(repo)

	// Test 1: Callback exchanges code, sets HttpOnly cookie, provisions profile
	req := httptest.NewRequest("GET", "/api/v1/auth/callback?code=testcode123&mock_user_id=dev-user-1", nil)
	w := httptest.NewRecorder()
	handler.HandleCallback(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.CookieName {
			sessionCookie = c
			break
		}
	}

	if sessionCookie == nil {
		t.Fatalf("expected session_token cookie to be set")
	}
	if !sessionCookie.HttpOnly {
		t.Errorf("expected cookie to be HttpOnly")
	}

	// Test 2: Authenticated request to /api/v1/auth/me using session cookie
	reqMe := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqMe.AddCookie(sessionCookie)
	wMe := httptest.NewRecorder()

	handler.RequireAuth(handler.HandleMe)(wMe, reqMe)
	respMe := wMe.Result()
	if respMe.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for /me, got %d", respMe.StatusCode)
	}

	var userCfg db.UserConfig
	if err := json.NewDecoder(respMe.Body).Decode(&userCfg); err != nil {
		t.Fatalf("failed to decode user config: %v", err)
	}
	if userCfg.UserID != "dev-user-1" {
		t.Errorf("expected UserID dev-user-1, got %s", userCfg.UserID)
	}
}
