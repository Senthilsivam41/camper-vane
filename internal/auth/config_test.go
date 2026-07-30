package auth

import (
	"testing"
)

func TestInitFromEnvProductionRequiresJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	if err := InitFromEnv(); err == nil {
		t.Fatal("expected error when production JWT_SECRET is missing")
	}

	t.Setenv("JWT_SECRET", defaultDevJWTSecret)
	if err := InitFromEnv(); err == nil {
		t.Fatal("expected error when production JWT_SECRET is the default dev secret")
	}

	t.Setenv("JWT_SECRET", "production-grade-secret-value-32b!")
	t.Setenv("COOKIE_SECURE", "")
	if err := InitFromEnv(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !CookieSecure() {
		t.Error("expected Secure cookies enabled by default in production")
	}
	if !IsProduction() {
		t.Error("expected IsProduction true")
	}
}

func TestInitFromEnvDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("COOKIE_SECURE", "false")
	if err := InitFromEnv(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if CookieSecure() {
		t.Error("expected Secure cookies disabled when COOKIE_SECURE=false")
	}
}
