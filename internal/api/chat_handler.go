package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"camper-vane/internal/db"
	"camper-vane/internal/proxy"
)

type ChatStreamHandler struct {
	userRepo    db.UserRepository
	sessionRepo db.SessionRepository
}

func NewChatStreamHandler(userRepo db.UserRepository, sessionRepo db.SessionRepository) *ChatStreamHandler {
	return &ChatStreamHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

type ChatStreamRequest struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
}

type MetricsEvent struct {
	SelectedModel      string `json:"selected_model"`
	RoutingRationale   string `json:"routing_rationale"`
	EstimatedCostDelta string `json:"estimated_cost_delta"`
}

type TextEvent struct {
	TextDelta string `json:"text_delta"`
}

type FinalUsageEvent struct {
	InputTokensConsumed int64 `json:"input_tokens_consumed"`
	OutputTokensConsumed int64 `json:"output_tokens_consumed"`
	UpdatedDailyTotal   int64 `json:"updated_daily_total"`
}

func (h *ChatStreamHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok || userID == "" {
		userID = "anonymous"
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	var req ChatStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt required", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		req.SessionID = "default-session"
	}

	if req.Model == "" {
		req.Model = "gemini-1.5-flash"
	}

	// 1. Fetch user config
	userCfg, err := h.userRepo.GetUserConfig(r.Context(), userID)
	if err != nil {
		userCfg = &db.UserConfig{DailyTokenCap: 50000, RoutingStrategy: "simple"}
	}

	// Set SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// 2. Emit `event: metrics`
	metrics := MetricsEvent{
		SelectedModel:      req.Model,
		RoutingRationale:   fmt.Sprintf("Negotiated downstream model %s under %s strategy.", req.Model, userCfg.RoutingStrategy),
		EstimatedCostDelta: "-$0.0015",
	}
	sendSSEEvent(w, flusher, "metrics", metrics)

	// Append user prompt to session store
	_ = h.sessionRepo.AppendToSession(r.Context(), req.SessionID, db.SessionMessage{
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Prompt,
		Timestamp: time.Now(),
	})

	// 3. Negotiate provider client and stream text deltas
	client := proxy.GetProviderClient(req.Model)
	chunkChan := make(chan proxy.StreamChunk, 100)

	proxyReq := proxy.ChatRequest{
		SessionID: req.SessionID,
		Prompt:    req.Prompt,
		Model:     req.Model,
	}

	go func() {
		defer close(chunkChan)
		_ = client.StreamChat(r.Context(), proxyReq, chunkChan)
	}()

	var fullAssistantText string
	var inputTokens, outputTokens int64

	for chunk := range chunkChan {
		if chunk.IsFinal {
			inputTokens = chunk.InputTokensConsumed
			outputTokens = chunk.OutputTokensConsumed
		} else if chunk.TextDelta != "" {
			fullAssistantText += chunk.TextDelta
			sendSSEEvent(w, flusher, "text", TextEvent{TextDelta: chunk.TextDelta})
		}
	}

	// Append assistant response to session store
	_ = h.sessionRepo.AppendToSession(r.Context(), req.SessionID, db.SessionMessage{
		SessionID: req.SessionID,
		Role:      "assistant",
		Content:   fullAssistantText,
		Timestamp: time.Now(),
	})

	// Increment daily usage
	totalTokens := inputTokens + outputTokens
	now := time.Now()
	_ = h.userRepo.IncrementDailyUsage(r.Context(), userID, now, totalTokens)
	updatedDaily, _ := h.userRepo.GetDailyUsage(r.Context(), userID, now)

	// 4. Emit `event: final_usage`
	finalUsage := FinalUsageEvent{
		InputTokensConsumed: inputTokens,
		OutputTokensConsumed: outputTokens,
		UpdatedDailyTotal:   updatedDaily,
	}
	sendSSEEvent(w, flusher, "final_usage", finalUsage)
}

func sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, eventName string, data interface{}) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, string(b))
	flusher.Flush()
}
