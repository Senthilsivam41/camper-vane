package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

const (
	OAuthStateCookie = "oauth_state"
	defaultFrontend  = "http://localhost:5173"
	defaultRedirect  = "http://localhost:5173/api/v1/auth/callback"

	defaultGoogleUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	defaultGitHubUserURL     = "https://api.github.com/user"
	defaultGitHubEmailURL    = "https://api.github.com/user/emails"
)

type OAuthIdentity struct {
	Provider string
	UserID   string
	Email    string
	Name     string
}

type OAuthManager struct {
	googleConfig *oauth2.Config
	githubConfig *oauth2.Config
	frontendURL  string
	redirectURI  string
	allowMock    bool

	httpClient          *http.Client
	googleUserInfoURL   string
	githubUserURL       string
	githubEmailURL      string
}

func NewOAuthManagerFromEnv() *OAuthManager {
	frontendURL := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
	if frontendURL == "" {
		frontendURL = defaultFrontend
	}
	redirectURI := strings.TrimSpace(os.Getenv("OAUTH_REDIRECT_URI"))
	if redirectURI == "" {
		redirectURI = defaultRedirect
	}

	m := &OAuthManager{
		frontendURL:       strings.TrimRight(frontendURL, "/"),
		redirectURI:       redirectURI,
		googleUserInfoURL: defaultGoogleUserInfoURL,
		githubUserURL:     defaultGitHubUserURL,
		githubEmailURL:    defaultGitHubEmailURL,
	}

	if cid, secret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"); cid != "" && secret != "" {
		m.googleConfig = &oauth2.Config{
			ClientID:     cid,
			ClientSecret: secret,
			RedirectURL:  redirectURI,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}
	}

	if cid, secret := os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"); cid != "" && secret != "" {
		m.githubConfig = &oauth2.Config{
			ClientID:     cid,
			ClientSecret: secret,
			RedirectURL:  redirectURI,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     github.Endpoint,
		}
	}

	mockFlag := strings.ToLower(strings.TrimSpace(os.Getenv("ALLOW_MOCK_AUTH")))
	switch mockFlag {
	case "true", "1", "yes":
		m.allowMock = true
	case "false", "0", "no":
		m.allowMock = false
	default:
		// Dev convenience: allow mock when no IdP credentials are configured.
		m.allowMock = !IsProduction() && m.googleConfig == nil && m.githubConfig == nil
	}

	return m
}

func (m *OAuthManager) client() *http.Client {
	if m.httpClient != nil {
		return m.httpClient
	}
	return http.DefaultClient
}

func (m *OAuthManager) AllowMock() bool {
	return m.allowMock
}

func (m *OAuthManager) FrontendURL() string {
	return m.frontendURL
}

func (m *OAuthManager) HasProvider(provider string) bool {
	switch strings.ToLower(provider) {
	case "google":
		return m.googleConfig != nil
	case "github":
		return m.githubConfig != nil
	default:
		return false
	}
}

func (m *OAuthManager) configFor(provider string) (*oauth2.Config, error) {
	switch strings.ToLower(provider) {
	case "google":
		if m.googleConfig == nil {
			return nil, fmt.Errorf("google OAuth is not configured (set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET)")
		}
		return m.googleConfig, nil
	case "github":
		if m.githubConfig == nil {
			return nil, fmt.Errorf("github OAuth is not configured (set GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET)")
		}
		return m.githubConfig, nil
	default:
		return nil, fmt.Errorf("unsupported OAuth provider: %s", provider)
	}
}

func (m *OAuthManager) AuthCodeURL(provider, state string) (string, error) {
	cfg, err := m.configFor(provider)
	if err != nil {
		return "", err
	}
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

func (m *OAuthManager) MockAuthURL(provider string) string {
	q := url.Values{}
	q.Set("code", "mock_code_123")
	q.Set("provider", provider)
	q.Set("mock_user_id", "developer-1")
	q.Set("format", "json")
	return "/api/v1/auth/callback?" + q.Encode()
}

func (m *OAuthManager) Exchange(ctx context.Context, provider, code string) (*OAuthIdentity, error) {
	cfg, err := m.configFor(provider)
	if err != nil {
		return nil, err
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, m.client())
	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	switch strings.ToLower(provider) {
	case "google":
		return m.fetchGoogleIdentity(ctx, token)
	case "github":
		return m.fetchGitHubIdentity(ctx, token)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (m *OAuthManager) fetchGoogleIdentity(ctx context.Context, token *oauth2.Token) (*OAuthIdentity, error) {
	userInfoURL := m.googleUserInfoURL
	if userInfoURL == "" {
		userInfoURL = defaultGoogleUserInfoURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := m.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo error (%d): %s", resp.StatusCode, string(b))
	}

	var info struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.ID == "" {
		return nil, fmt.Errorf("google userinfo missing id")
	}

	email := info.Email
	if email == "" {
		email = info.ID + "@users.noreply.google.com"
	}

	return &OAuthIdentity{
		Provider: "google",
		UserID:   "google:" + info.ID,
		Email:    email,
		Name:     info.Name,
	}, nil
}

func (m *OAuthManager) fetchGitHubIdentity(ctx context.Context, token *oauth2.Token) (*OAuthIdentity, error) {
	userURL := m.githubUserURL
	if userURL == "" {
		userURL = defaultGitHubUserURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := m.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user error (%d): %s", resp.StatusCode, string(b))
	}

	var info struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.ID == 0 {
		return nil, fmt.Errorf("github user missing id")
	}

	email := info.Email
	if email == "" {
		email, _ = m.fetchGitHubPrimaryEmail(ctx, token)
	}
	if email == "" {
		email = fmt.Sprintf("%s@users.noreply.github.com", info.Login)
	}

	name := info.Name
	if name == "" {
		name = info.Login
	}

	return &OAuthIdentity{
		Provider: "github",
		UserID:   fmt.Sprintf("github:%d", info.ID),
		Email:    email,
		Name:     name,
	}, nil
}

func (m *OAuthManager) fetchGitHubPrimaryEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	emailURL := m.githubEmailURL
	if emailURL == "" {
		emailURL = defaultGitHubEmailURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, emailURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := m.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails error (%d)", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", nil
}

func SetOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookie,
		Value:    state,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     OAuthStateCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
