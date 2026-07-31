package proxy

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var allowMockProviders = true // default for tests/local until InitCredentialsFromEnv runs

// Credential model (P0-4 / Epic 2):
//   - Server-held secrets only: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, PERPLEXITY_API_KEY.
//   - UI never collects provider keys (zero-trust toward end users).
//   - Development: missing keys → MockClient stream.
//   - Production (or ALLOW_MOCK_PROVIDERS=false): missing keys → hard error (no silent mock).

// InitCredentialsFromEnv configures whether missing provider API keys may fall back to MockClient.
func InitCredentialsFromEnv() {
	flag := strings.ToLower(strings.TrimSpace(os.Getenv("ALLOW_MOCK_PROVIDERS")))
	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "development"
	}

	switch flag {
	case "true", "1", "yes":
		allowMockProviders = true
	case "false", "0", "no":
		allowMockProviders = false
	default:
		allowMockProviders = appEnv != "production"
	}

	log.Printf("proxy: allow_mock_providers=%v (APP_ENV=%s)", allowMockProviders, appEnv)
}

func AllowMockProviders() bool {
	return allowMockProviders
}

// ResolveAPIKey returns the env value for keyName, or an error when missing and mocks are disallowed.
func ResolveAPIKey(envName string) (string, error) {
	key := strings.TrimSpace(os.Getenv(envName))
	if key != "" {
		return key, nil
	}
	if allowMockProviders {
		return "", nil
	}
	return "", fmt.Errorf("%s is required (set server-side provider key or ALLOW_MOCK_PROVIDERS=true for local mock streams)", envName)
}
