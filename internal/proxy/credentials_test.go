package proxy

import (
	"strings"
	"testing"
)

func TestResolveAPIKeyFailLoud(t *testing.T) {
	t.Setenv("ALLOW_MOCK_PROVIDERS", "false")
	t.Setenv("APP_ENV", "production")
	InitCredentialsFromEnv()

	t.Setenv("OPENAI_API_KEY", "")
	_, err := ResolveAPIKey("OPENAI_API_KEY")
	if err == nil {
		t.Fatal("expected error when provider key missing and mocks disabled")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveAPIKeyAllowsMock(t *testing.T) {
	t.Setenv("ALLOW_MOCK_PROVIDERS", "true")
	t.Setenv("APP_ENV", "development")
	InitCredentialsFromEnv()

	t.Setenv("OPENAI_API_KEY", "")
	key, err := ResolveAPIKey("OPENAI_API_KEY")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "" {
		t.Fatalf("expected empty key for mock path, got %q", key)
	}
}
