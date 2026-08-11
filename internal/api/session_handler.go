package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"camper-vane/internal/db"
)

type SessionHandler struct {
	sessionRepo db.SessionRepository
}

func NewSessionHandler(sessionRepo db.SessionRepository) *SessionHandler {
	return &SessionHandler{sessionRepo: sessionRepo}
}

func (h *SessionHandler) HandleSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions")
	path = strings.Trim(path, "/")

	switch {
	case path == "" && r.Method == http.MethodGet:
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		sessions, err := h.sessionRepo.ListSessions(r.Context(), userID, limit)
		if err != nil {
			http.Error(w, "Failed to list sessions: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if sessions == nil {
			sessions = []db.SessionSummary{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sessions)

	case strings.HasSuffix(path, "/messages") && r.Method == http.MethodGet:
		sessionID := strings.TrimSuffix(path, "/messages")
		sessionID = strings.Trim(sessionID, "/")
		if sessionID == "" {
			http.Error(w, "session id required", http.StatusBadRequest)
			return
		}
		limit := 100
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				limit = n
			}
		}
		msgs, err := h.sessionRepo.GetSessionHistory(r.Context(), sessionID, limit)
		if err != nil {
			http.Error(w, "Failed to load session history: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Enforce ownership when user_id is present on messages.
		filtered := make([]db.SessionMessage, 0, len(msgs))
		for _, m := range msgs {
			if m.UserID == "" || m.UserID == userID {
				filtered = append(filtered, m)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(filtered)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

type UsageHandler struct {
	userRepo db.UserRepository
}

func NewUsageHandler(userRepo db.UserRepository) *UsageHandler {
	return &UsageHandler{userRepo: userRepo}
}

type UsageResponse struct {
	UserID          string `json:"user_id"`
	WindowHours     int    `json:"window_hours"`
	TokensUsed      int64  `json:"tokens_used"`
	DailyTokenCap   int64  `json:"daily_token_cap"`
	UtilizationPct  float64 `json:"utilization_pct"`
	AsOf            string `json:"as_of"`
}

func (h *UsageHandler) HandleUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	cfg, err := h.userRepo.GetUserConfig(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to load user config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	used, err := h.userRepo.GetUsageSince(r.Context(), userID, since)
	if err != nil {
		http.Error(w, "Failed to load usage: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var pct float64
	if cfg.DailyTokenCap > 0 {
		pct = (float64(used) / float64(cfg.DailyTokenCap)) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(UsageResponse{
		UserID:         userID,
		WindowHours:    24,
		TokensUsed:     used,
		DailyTokenCap:  cfg.DailyTokenCap,
		UtilizationPct: pct,
		AsOf:           time.Now().UTC().Format(time.RFC3339),
	})
}
