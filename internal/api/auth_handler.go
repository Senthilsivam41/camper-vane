package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"camper-vane/internal/auth"
	"camper-vane/internal/db"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

type AuthHandler struct {
	userRepo db.UserRepository
}

func NewAuthHandler(userRepo db.UserRepository) *AuthHandler {
	return &AuthHandler{userRepo: userRepo}
}

type LoginResponse struct {
	AuthURL string `json:"auth_url"`
}

type AuthCallbackRequest struct {
	Code     string `json:"code"`
	Provider string `json:"provider"`
	// Dev/demo bypass option for quick testing
	MockUserID string `json:"mock_user_id,omitempty"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "google"
	}

	// Returns OAuth auth URL or mock URL for local dev
	resp := LoginResponse{
		AuthURL: "/api/v1/auth/callback?code=mock_code_123&provider=" + provider,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		// Try parsing JSON body if POST
		var req AuthCallbackRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			code = req.Code
		}
	}

	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	// Extract or mock identity
	userID := r.URL.Query().Get("mock_user_id")
	if userID == "" {
		userID = "user-" + code
	}
	email := userID + "@camper-vane.local"

	// Provision profile in database if new user
	ctx := r.Context()
	userCfg, err := h.userRepo.GetUserConfig(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to provision user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate 24h JWT
	tokenString, err := auth.GenerateToken(userID, email, 24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to create session token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set HttpOnly, Secure cookie
	auth.SetSessionCookie(w, tokenString, 24*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "authenticated",
		"user":    userCfg,
		"message": "Session established via HttpOnly cookie",
	})
}

func (h *AuthHandler) HandleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userCfg, err := h.userRepo.GetUserConfig(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch user config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userCfg)
}

func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	auth.ClearSessionCookie(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "logged_out"})
}

func (h *AuthHandler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized: missing session cookie", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized: invalid session token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
