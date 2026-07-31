package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewOAuthManagerMockDefault(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_MOCK_AUTH", "")
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	_ = InitFromEnv()

	m := NewOAuthManagerFromEnv()
	if !m.AllowMock() {
		t.Fatal("expected mock auth when no IdP credentials configured")
	}
	if m.HasProvider("google") || m.HasProvider("github") {
		t.Fatal("expected no real providers without credentials")
	}
	url := m.MockAuthURL("google")
	if !strings.Contains(url, "mock_user_id=developer-1") {
		t.Fatalf("unexpected mock URL: %s", url)
	}
}

func TestNewOAuthManagerGoogleConfigured(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_MOCK_AUTH", "false")
	t.Setenv("GOOGLE_CLIENT_ID", "gid")
	t.Setenv("GOOGLE_CLIENT_SECRET", "gsecret")
	t.Setenv("GITHUB_CLIENT_ID", "")
	t.Setenv("GITHUB_CLIENT_SECRET", "")
	_ = InitFromEnv()

	m := NewOAuthManagerFromEnv()
	if !m.HasProvider("google") {
		t.Fatal("expected google provider")
	}
	if m.AllowMock() {
		t.Fatal("expected mock disabled when ALLOW_MOCK_AUTH=false")
	}
	authURL, err := m.AuthCodeURL("google", "state123")
	if err != nil {
		t.Fatalf("AuthCodeURL: %v", err)
	}
	if !strings.Contains(authURL, "state123") {
		t.Fatalf("auth URL missing state: %s", authURL)
	}
}

func TestExchangeGoogle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"atok","token_type":"Bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer atok" {
			t.Fatalf("expected bearer token, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":    "99",
			"email": "dev@example.com",
			"name":  "Dev",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m := &OAuthManager{
		httpClient:        srv.Client(),
		googleUserInfoURL: srv.URL + "/userinfo",
		googleConfig: &oauth2.Config{
			ClientID:     "id",
			ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  srv.URL + "/auth",
				TokenURL: srv.URL + "/token",
			},
		},
	}

	id, err := m.Exchange(context.Background(), "google", "auth-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if id.UserID != "google:99" || id.Email != "dev@example.com" {
		t.Fatalf("unexpected identity: %+v", id)
	}
}

func TestExchangeGitHubFallsBackToEmails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"gh-tok","token_type":"bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    42,
			"login": "octo",
			"name":  "",
			"email": "",
		})
	})
	mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"email": "octo@users.noreply.github.com", "primary": true, "verified": true},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	m := &OAuthManager{
		httpClient:     srv.Client(),
		githubUserURL:  srv.URL + "/user",
		githubEmailURL: srv.URL + "/emails",
		githubConfig: &oauth2.Config{
			ClientID:     "id",
			ClientSecret: "secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  srv.URL + "/auth",
				TokenURL: srv.URL + "/token",
			},
		},
	}

	id, err := m.Exchange(context.Background(), "github", "auth-code")
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}
	if id.UserID != "github:42" {
		t.Fatalf("unexpected user id: %s", id.UserID)
	}
	if id.Email != "octo@users.noreply.github.com" {
		t.Fatalf("unexpected email: %s", id.Email)
	}
	if id.Name != "octo" {
		t.Fatalf("expected login fallback name, got %s", id.Name)
	}
}

func TestSessionCookieRespectsSecureFlag(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "production-grade-secret-value-32b!")
	t.Setenv("COOKIE_SECURE", "")
	if err := InitFromEnv(); err != nil {
		t.Fatalf("InitFromEnv: %v", err)
	}

	rec := httptest.NewRecorder()
	SetSessionCookie(rec, "tok", time.Hour)
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}
	if !cookies[0].HttpOnly {
		t.Error("expected HttpOnly")
	}
	if !cookies[0].Secure {
		t.Error("expected Secure cookie in production")
	}

	stateRec := httptest.NewRecorder()
	SetOAuthStateCookie(stateRec, "google:abc")
	stateCookies := stateRec.Result().Cookies()
	if len(stateCookies) == 0 || !stateCookies[0].Secure || !stateCookies[0].HttpOnly {
		t.Fatalf("expected Secure HttpOnly oauth_state cookie, got %+v", stateCookies)
	}
}
