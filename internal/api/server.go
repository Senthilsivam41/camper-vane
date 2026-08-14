package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"camper-vane/internal/auth"
	"camper-vane/internal/db"
	"camper-vane/internal/router"
)

// ServerDeps wires handlers for the HTTP API surface.
type ServerDeps struct {
	Store db.Store
	OAuth *auth.OAuthManager
}

// NewHTTPHandler builds the authenticated API mux, optionally wrapped with CORS.
func NewHTTPHandler(deps ServerDeps) http.Handler {
	oauth := deps.OAuth
	if oauth == nil {
		oauth = auth.NewOAuthManagerFromEnv()
	}

	routerEngine := router.NewRouter(deps.Store, deps.Store)
	authHandler := NewAuthHandler(deps.Store, oauth)
	userHandler := NewUserHandler(deps.Store)
	chatHandler := NewChatStreamHandler(deps.Store, deps.Store, routerEngine)
	sessionHandler := NewSessionHandler(deps.Store)
	usageHandler := NewUsageHandler(deps.Store)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/auth/login", authHandler.HandleLogin)
	mux.HandleFunc("/api/v1/auth/callback", authHandler.HandleCallback)
	mux.HandleFunc("/api/v1/auth/me", authHandler.RequireAuth(authHandler.HandleMe))
	mux.HandleFunc("/api/v1/auth/logout", authHandler.HandleLogout)
	mux.HandleFunc("/api/v1/user/config", authHandler.RequireAuth(userHandler.HandleUserConfig))
	mux.HandleFunc("/api/v1/user/usage", authHandler.RequireAuth(usageHandler.HandleUsage))
	mux.HandleFunc("/api/v1/sessions", authHandler.RequireAuth(sessionHandler.HandleSessions))
	mux.HandleFunc("/api/v1/sessions/", authHandler.RequireAuth(sessionHandler.HandleSessions))
	mux.HandleFunc("/api/v1/chat/stream", authHandler.RequireAuth(chatHandler.HandleStream))

	return CORSMiddleware(mux)
}

// CORSMiddleware enables credentialed cross-origin access when CORS_ALLOWED_ORIGINS is set.
// Empty/unset → pass-through (same-origin / reverse-proxy recommended).
// Wildcard "*" is rejected at boot (config error): cookie auth requires exact origins.
func CORSMiddleware(next http.Handler) http.Handler {
	allowed, err := parseCORSOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if err != nil {
		log.Fatalf("invalid CORS_ALLOWED_ORIGINS: %v", err)
	}
	if len(allowed) == 0 {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && originAllowed(allowed, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func parseCORSOrigins(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			return nil, fmt.Errorf("wildcard '*' is not allowed; list exact origins only (empty disables CORS)")
		}
		out = append(out, p)
	}
	return out, nil
}

func originAllowed(allowed []string, origin string) bool {
	for _, a := range allowed {
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}
