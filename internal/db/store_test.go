package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRepositoryContractMemory(t *testing.T) {
	exerciseStoreContract(t, NewMemoryRepo())
}

func TestRepositoryContractSQLite(t *testing.T) {
	repo, err := NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("sqlite init: %v", err)
	}
	defer repo.Close()
	exerciseStoreContract(t, repo)
}

func TestNewStoreFromEnvUsesSQLiteByDefault(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_PATH", ":memory:")
	store, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv: %v", err)
	}
	defer store.Close()
	if _, ok := store.(*SQLiteRepo); !ok {
		t.Fatalf("expected *SQLiteRepo, got %T", store)
	}
}

func TestNewStoreFromEnvPostgresRequiresReachableURL(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live postgres factory test")
	}
	t.Setenv("DATABASE_URL", url)
	store, err := NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv postgres: %v", err)
	}
	defer store.Close()
	if _, ok := store.(*PostgresRepo); !ok {
		t.Fatalf("expected *PostgresRepo, got %T", store)
	}
	exerciseStoreContract(t, store)
}

func exerciseStoreContract(t *testing.T, store Store) {
	t.Helper()
	ctx := context.Background()

	cfg, err := store.GetUserConfig(ctx, "contract-user")
	if err != nil {
		t.Fatalf("GetUserConfig: %v", err)
	}
	if cfg.DailyTokenCap != 50000 {
		t.Fatalf("expected default cap 50000, got %d", cfg.DailyTokenCap)
	}

	cfg.DailyTokenCap = 75000
	cfg.RoutingStrategy = "advanced"
	cfg.PreferredModels = []string{"sonar", "gpt-4o-mini"}
	if err := store.UpdateUserConfig(ctx, cfg); err != nil {
		t.Fatalf("UpdateUserConfig: %v", err)
	}

	updated, err := store.GetUserConfig(ctx, "contract-user")
	if err != nil {
		t.Fatalf("GetUserConfig reload: %v", err)
	}
	if updated.DailyTokenCap != 75000 || updated.RoutingStrategy != "advanced" {
		t.Fatalf("unexpected config: %+v", updated)
	}

	now := time.Now()
	if err := store.IncrementDailyUsage(ctx, "contract-user", now, 120); err != nil {
		t.Fatalf("IncrementDailyUsage: %v", err)
	}
	if err := store.IncrementDailyUsage(ctx, "contract-user", now, 30); err != nil {
		t.Fatalf("IncrementDailyUsage second: %v", err)
	}
	usage, err := store.GetDailyUsage(ctx, "contract-user", now)
	if err != nil || usage != 150 {
		t.Fatalf("expected usage 150, got %d err=%v", usage, err)
	}

	sess := "contract-sess"
	if err := store.AppendToSession(ctx, sess, SessionMessage{
		Role: "user", Content: "one", Timestamp: now,
	}); err != nil {
		t.Fatalf("AppendToSession: %v", err)
	}
	if err := store.AppendToSession(ctx, sess, SessionMessage{
		Role: "assistant", Content: "two", Timestamp: now.Add(time.Second),
	}); err != nil {
		t.Fatalf("AppendToSession 2: %v", err)
	}

	hist, err := store.GetSessionHistory(ctx, sess, 5)
	if err != nil || len(hist) != 2 {
		t.Fatalf("expected 2 history msgs, got %d err=%v", len(hist), err)
	}
	if hist[0].Content != "one" || hist[1].Content != "two" {
		t.Fatalf("history not chronological: %+v", hist)
	}
}
