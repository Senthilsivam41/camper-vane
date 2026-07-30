package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"camper-vane/internal/auth"
	"camper-vane/internal/db"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

type AuthHandler struct {
	userRepo db.UserRepository
	oauth    *auth.OAuthManager
}

func NewAuthHandler(userRepo db.UserRepository, oauth *auth.OAuthManager) *AuthHandler {
	if oauth == nil {
		oauth = auth.NewOAuthManagerFromEnv()
	}
	return &AuthHandler{userRepo: userRepo, oauth: oauth}
}

type LoginResponse struct {
	AuthURL    string   `json:"auth_url"`
	Provider   string   `json:"provider"`
	Mock       bool     `json:"mock"`
	Providers  []string `json:"available_providers"`
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
	provider = strings.ToLower(provider)

	available := h.availableProviders()

	// Status/probe: no state cookie side effects.
	if r.URL.Query().Get("intent") == "status" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Provider:  provider,
			Mock:      h.oauth.AllowMock(),
			Providers: available,
			AuthURL:   "",
		})
		return
	}

	// Real OAuth when credentials exist for the requested provider.
	if h.oauth.HasProvider(provider) {
		state, err := randomState()
		if err != nil {
			http.Error(w, "Failed to create OAuth state", http.StatusInternalServerError)
			return
		}
		authURL, err := h.oauth.AuthCodeURL(provider, state)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		auth.SetOAuthStateCookie(w, provider+":"+state)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(LoginResponse{
			AuthURL:   authURL,
			Provider:  provider,
			Mock:      false,
			Providers: available,
		})
		return
	}

	if !h.oauth.AllowMock() {
		http.Error(w, "OAuth provider not configured: "+provider, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(LoginResponse{
		AuthURL:   h.oauth.MockAuthURL(provider),
		Provider:  provider,
		Mock:      true,
		Providers: available,
	})
}

func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	code := q.Get("code")
	provider := strings.ToLower(q.Get("provider"))
	mockUserID := q.Get("mock_user_id")
	state := q.Get("state")
	format := q.Get("format")

	if r.Method == http.MethodPost && code == "" {
		var req struct {
			Code       string `json:"code"`
			Provider   string `json:"provider"`
			MockUserID string `json:"mock_user_id"`
			State      string `json:"state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			code = req.Code
			if provider == "" {
				provider = strings.ToLower(req.Provider)
			}
			if mockUserID == "" {
				mockUserID = req.MockUserID
			}
			if state == "" {
				state = req.State
			}
		}
	}

	if code == "" {
		http.Error(w, "Missing code parameter", http.StatusBadRequest)
		return
	}

	var (
		userID string
		email  string
	)

	// Mock auth path (local/dev only).
	if mockUserID != "" || strings.HasPrefix(code, "mock_") {
		if !h.oauth.AllowMock() {
			http.Error(w, "Mock authentication is disabled", http.StatusForbidden)
			return
		}
		userID = mockUserID
		if userID == "" {
			userID = "user-" + code
		}
		email = userID + "@camper-vane.local"
		if provider == "" {
			provider = "mock"
		}
	} else {
		// Real OAuth: validate state and exchange code.
		if provider == "" {
			// Infer provider from oauth_state cookie (provider:state).
			if c, err := r.Cookie(auth.OAuthStateCookie); err == nil && c.Value != "" {
				parts := strings.SplitN(c.Value, ":", 2)
				if len(parts) == 2 {
					provider = parts[0]
					if state == "" {
						state = parts[1]
					}
				}
			}
		}
		if provider == "" {
			http.Error(w, "Missing provider parameter", http.StatusBadRequest)
			return
		}
		if err := h.validateOAuthState(r, provider, state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		auth.ClearOAuthStateCookie(w)

		identity, err := h.oauth.Exchange(r.Context(), provider, code)
		if err != nil {
			http.Error(w, "OAuth exchange failed: "+err.Error(), http.StatusUnauthorized)
			return
		}
		userID = identity.UserID
		email = identity.Email
	}

	ctx := r.Context()
	userCfg, err := h.userRepo.GetUserConfig(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to provision user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := auth.GenerateToken(userID, email, 24*time.Hour)
	if err != nil {
		http.Error(w, "Failed to create session token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	auth.SetSessionCookie(w, tokenString, 24*time.Hour)

	isMockAuth := mockUserID != "" || strings.HasPrefix(code, "mock_")
	wantsJSON := isMockAuth ||
		format == "json" ||
		strings.Contains(r.Header.Get("Accept"), "application/json") ||
		r.Header.Get("X-Requested-With") == "XMLHttpRequest"

	if !wantsJSON {
		http.Redirect(w, r, h.oauth.FrontendURL()+"/?auth=success", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "authenticated",
		"user":     userCfg,
		"provider": provider,
		"message":  "Session established via HttpOnly cookie",
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

func (h *AuthHandler) availableProviders() []string {
	var out []string
	if h.oauth.HasProvider("google") {
		out = append(out, "google")
	}
	if h.oauth.HasProvider("github") {
		out = append(out, "github")
	}
	if h.oauth.AllowMock() {
		out = append(out, "mock")
	}
	return out
}

func (h *AuthHandler) validateOAuthState(r *http.Request, provider, state string) error {
	if state == "" {
		return errInvalidState
	}
	c, err := r.Cookie(auth.OAuthStateCookie)
	if err != nil || c.Value == "" {
		return errInvalidState
	}
	expected := provider + ":" + state
	if c.Value != expected {
		// Also accept raw state match if cookie stores only state.
		if c.Value != state && !strings.HasSuffix(c.Value, ":"+state) {
			return errInvalidState
		}
	}
	return nil
}

var errInvalidState = &oauthStateError{msg: "invalid or missing OAuth state"}

type oauthStateError struct{ msg string }

func (e *oauthStateError) Error() string { return e.msg }

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
