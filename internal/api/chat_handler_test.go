package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"camper-vane/internal/db"
)

func TestChatStreamHandler(t *testing.T) {
	repo, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	streamHandler := NewChatStreamHandler(repo, repo)
	authHandler := NewAuthHandler(repo)

	// Register user session
	wAuth := httptest.NewRecorder()
	reqAuth := httptest.NewRequest("GET", "/api/v1/auth/callback?code=testcode&mock_user_id=stream-user-1", nil)
	authHandler.HandleCallback(wAuth, reqAuth)
	cookie := wAuth.Result().Cookies()[0]

	payload, _ := json.Marshal(ChatStreamRequest{
		SessionID: "test-session-1",
		Prompt:    "Write a hello world function",
		Model:     "gemini-1.5-flash",
	})

	req := httptest.NewRequest("POST", "/api/v1/chat/stream", bytes.NewBuffer(payload))
	req.AddCookie(cookie)
	w := httptest.NewRecorder()

	authHandler.RequireAuth(streamHandler.HandleStream)(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}

	bodyStr := w.Body.String()

	if !strings.Contains(bodyStr, "event: metrics") {
		t.Errorf("expected SSE event: metrics in body")
	}
	if !strings.Contains(bodyStr, "event: text") {
		t.Errorf("expected SSE event: text in body")
	}
	if !strings.Contains(bodyStr, "event: final_usage") {
		t.Errorf("expected SSE event: final_usage in body")
	}

	// Verify session history saved
	history, err := repo.GetSessionHistory(req.Context(), "test-session-1", 10)
	if err != nil || len(history) < 2 {
		t.Fatalf("expected at least 2 session messages (user + assistant), got %d", len(history))
	}
}
