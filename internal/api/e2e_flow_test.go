package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"camper-vane/internal/auth"
	"camper-vane/internal/db"
	"camper-vane/internal/proxy"
)

// TestE2EHappyPath covers login → config → usage → stream → sessions → logout.
func TestE2EHappyPath(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("ALLOW_MOCK_AUTH", "true")
	t.Setenv("ALLOW_MOCK_PROVIDERS", "true")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	t.Setenv("COOKIE_SECURE", "false")
	t.Setenv("COOKIE_SAMESITE", "lax")

	if err := auth.InitFromEnv(); err != nil {
		t.Fatalf("auth init: %v", err)
	}
	proxy.InitCredentialsFromEnv()

	store, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	defer store.Close()

	handler := NewHTTPHandler(ServerDeps{
		Store: store,
		OAuth: auth.NewOAuthManagerFromEnv(),
	})

	jar := &cookieJar{}

	// 1) Mock login
	loginReq := httptest.NewRequest(http.MethodGet,
		"/api/v1/auth/callback?code=mock_e2e&mock_user_id=e2e-user&format=json", nil)
	loginReq.Header.Set("Origin", "http://localhost:5173")
	loginReq.Header.Set("Accept", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRec.Code, loginRec.Body.String())
	}
	if got := loginRec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("CORS origin on login=%q", got)
	}
	jar.capture(loginRec.Result().Cookies())
	if jar.get(auth.CookieName) == "" {
		t.Fatal("expected session_token cookie after login")
	}

	// 2) /auth/me
	meRec := doAuthed(handler, jar, http.MethodGet, "/api/v1/auth/me", nil)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me status=%d", meRec.Code)
	}
	var me db.UserConfig
	if err := json.Unmarshal(meRec.Body.Bytes(), &me); err != nil || me.UserID != "e2e-user" {
		t.Fatalf("me decode: err=%v cfg=%+v", err, me)
	}

	// 3) Update config
	cfgBody := map[string]interface{}{
		"daily_token_cap":   60000,
		"routing_strategy":  "advanced",
		"preferred_models":  []string{"gemini-1.5-flash", "claude-3-5-sonnet"},
	}
	cfgBytes, _ := json.Marshal(cfgBody)
	cfgRec := doAuthed(handler, jar, http.MethodPut, "/api/v1/user/config", bytes.NewReader(cfgBytes))
	if cfgRec.Code != http.StatusOK {
		t.Fatalf("config put status=%d body=%s", cfgRec.Code, cfgRec.Body.String())
	}

	// 4) Usage before chat
	usageBeforeRec := doAuthed(handler, jar, http.MethodGet, "/api/v1/user/usage", nil)
	if usageBeforeRec.Code != http.StatusOK {
		t.Fatalf("usage status=%d", usageBeforeRec.Code)
	}
	var usageBefore UsageResponse
	_ = json.Unmarshal(usageBeforeRec.Body.Bytes(), &usageBefore)
	if usageBefore.DailyTokenCap != 60000 {
		t.Fatalf("expected cap 60000, got %d", usageBefore.DailyTokenCap)
	}

	// 5) Chat stream
	streamBody, _ := json.Marshal(ChatStreamRequest{
		SessionID: "e2e-session",
		Prompt:    "Hi, hello! What is 2 + 2?",
		Model:     "",
	})
	streamRec := doAuthed(handler, jar, http.MethodPost, "/api/v1/chat/stream", bytes.NewReader(streamBody))
	if streamRec.Code != http.StatusOK {
		t.Fatalf("stream status=%d body=%s", streamRec.Code, streamRec.Body.String())
	}
	streamOut := streamRec.Body.String()
	for _, evt := range []string{"event: metrics", "event: text", "event: final_usage"} {
		if !strings.Contains(streamOut, evt) {
			t.Fatalf("stream missing %s\n%s", evt, streamOut)
		}
	}
	if !strings.Contains(streamOut, `"selected_model"`) {
		t.Fatalf("metrics missing selected_model: %s", streamOut)
	}

	// 6) Usage after chat should increase
	usageAfterRec := doAuthed(handler, jar, http.MethodGet, "/api/v1/user/usage", nil)
	var usageAfter UsageResponse
	_ = json.Unmarshal(usageAfterRec.Body.Bytes(), &usageAfter)
	if usageAfter.TokensUsed <= usageBefore.TokensUsed {
		t.Fatalf("expected usage to increase: before=%d after=%d", usageBefore.TokensUsed, usageAfter.TokensUsed)
	}

	// 7) Sessions list + history
	sessRec := doAuthed(handler, jar, http.MethodGet, "/api/v1/sessions", nil)
	if sessRec.Code != http.StatusOK {
		t.Fatalf("sessions status=%d", sessRec.Code)
	}
	var sessions []db.SessionSummary
	if err := json.Unmarshal(sessRec.Body.Bytes(), &sessions); err != nil || len(sessions) == 0 {
		t.Fatalf("sessions: err=%v len=%d body=%s", err, len(sessions), sessRec.Body.String())
	}

	histRec := doAuthed(handler, jar, http.MethodGet, "/api/v1/sessions/e2e-session/messages", nil)
	if histRec.Code != http.StatusOK {
		t.Fatalf("history status=%d", histRec.Code)
	}
	var msgs []db.SessionMessage
	if err := json.Unmarshal(histRec.Body.Bytes(), &msgs); err != nil || len(msgs) < 2 {
		t.Fatalf("history: err=%v len=%d", err, len(msgs))
	}

	// 8) Logout clears session
	logoutRec := doAuthed(handler, jar, http.MethodPost, "/api/v1/auth/logout", nil)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status=%d", logoutRec.Code)
	}
	jar.capture(logoutRec.Result().Cookies())

	meAfter := doAuthed(handler, jar, http.MethodGet, "/api/v1/auth/me", nil)
	if meAfter.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized after logout, got %d body=%s", meAfter.Code, meAfter.Body.String())
	}
}

type cookieJar struct {
	cookies map[string]*http.Cookie
}

func (j *cookieJar) capture(cs []*http.Cookie) {
	if j.cookies == nil {
		j.cookies = map[string]*http.Cookie{}
	}
	for _, c := range cs {
		cp := *c
		j.cookies[c.Name] = &cp
	}
}

func (j *cookieJar) get(name string) string {
	if j.cookies == nil || j.cookies[name] == nil {
		return ""
	}
	return j.cookies[name].Value
}

func doAuthed(h http.Handler, jar *cookieJar, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Origin", "http://localhost:5173")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c := jar.cookies[auth.CookieName]; c != nil && c.Value != "" {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	jar.capture(rec.Result().Cookies())
	return rec
}
