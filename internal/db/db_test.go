package db

import (
	"context"
	"testing"
	"time"
)

func TestSQLiteRepo(t *testing.T) {
	ctx := context.Background()
	repo, err := NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	// Test GetUserConfig auto-provisioning
	cfg, err := repo.GetUserConfig(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error getting config: %v", err)
	}
	if cfg.UserID != "user-1" {
		t.Errorf("expected UserID user-1, got %s", cfg.UserID)
	}
	if cfg.DailyTokenCap != 50000 {
		t.Errorf("expected default DailyTokenCap 50000, got %d", cfg.DailyTokenCap)
	}

	// Test UpdateUserConfig
	cfg.DailyTokenCap = 100000
	cfg.RoutingStrategy = "advanced"
	cfg.PreferredModels = []string{"claude-3-5-sonnet", "gpt-4o"}
	if err := repo.UpdateUserConfig(ctx, cfg); err != nil {
		t.Fatalf("failed to update user config: %v", err)
	}

	updated, err := repo.GetUserConfig(ctx, "user-1")
	if err != nil {
		t.Fatalf("failed to fetch updated config: %v", err)
	}
	if updated.DailyTokenCap != 100000 || updated.RoutingStrategy != "advanced" {
		t.Errorf("config update mismatch: %+v", updated)
	}

	// Test Daily Usage
	now := time.Now()
	usage, err := repo.GetDailyUsage(ctx, "user-1", now)
	if err != nil || usage != 0 {
		t.Errorf("expected 0 initial usage, got %d, err: %v", usage, err)
	}

	if err := repo.IncrementDailyUsage(ctx, "user-1", now, 1500); err != nil {
		t.Fatalf("failed to increment usage: %v", err)
	}

	usageAfter, err := repo.GetDailyUsage(ctx, "user-1", now)
	if err != nil || usageAfter != 1500 {
		t.Errorf("expected 1500 usage, got %d, err: %v", usageAfter, err)
	}
}
