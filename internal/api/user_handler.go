package api

import (
	"encoding/json"
	"net/http"

	"camper-vane/internal/db"
)

type UserHandler struct {
	userRepo db.UserRepository
}

func NewUserHandler(userRepo db.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

type UpdateUserConfigRequest struct {
	DailyTokenCap   int64    `json:"daily_token_cap"`
	RoutingStrategy string   `json:"routing_strategy"`
	PreferredModels []string `json:"preferred_models"`
}

func (h *UserHandler) HandleUserConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		config, err := h.userRepo.GetUserConfig(r.Context(), userID)
		if err != nil {
			http.Error(w, "Failed to get user config: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(config)

	case http.MethodPut:
		var req UpdateUserConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Validation
		if req.DailyTokenCap < 0 {
			http.Error(w, "Validation error: daily_token_cap cannot be negative", http.StatusBadRequest)
			return
		}

		if req.RoutingStrategy != "simple" && req.RoutingStrategy != "advanced" {
			http.Error(w, "Validation error: routing_strategy must be 'simple' or 'advanced'", http.StatusBadRequest)
			return
		}

		config := &db.UserConfig{
			UserID:          userID,
			DailyTokenCap:   req.DailyTokenCap,
			RoutingStrategy: req.RoutingStrategy,
			PreferredModels: req.PreferredModels,
		}

		if err := h.userRepo.UpdateUserConfig(r.Context(), config); err != nil {
			http.Error(w, "Failed to update user config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(config)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
