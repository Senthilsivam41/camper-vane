package router

import (
	"context"
	"testing"
	"time"

	"camper-vane/internal/db"
)

func TestRouterSimpleModeThrottle(t *testing.T) {
	repo, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	router := NewRouter(repo, repo)

	// Set user config cap = 1000 tokens
	cfg := &db.UserConfig{
		UserID:          "user-budget",
		DailyTokenCap:   1000,
		RoutingStrategy: "advanced",
	}
	if err := repo.UpdateUserConfig(ctx, cfg); err != nil {
		t.Fatalf("failed to setup user config: %v", err)
	}

	// Increment usage to 880 tokens (88% utilization >= 85%)
	if err := repo.IncrementDailyUsage(ctx, "user-budget", time.Now(), 880); err != nil {
		t.Fatalf("failed to increment usage: %v", err)
	}

	decision, err := router.EvaluateRoute(ctx, RouteRequest{
		UserID:         "user-budget",
		SessionID:      "sess-b1",
		Prompt:         "Design a complex microservice architecture with deadlock prevention",
		RequestedModel: "claude-3-5-sonnet",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !decision.BudgetThrottled {
		t.Errorf("expected BudgetThrottled to be true when utilization >= 85%%")
	}
	if decision.SelectedModel != "gemini-1.5-flash" {
		t.Errorf("expected model to be throttled to gemini-1.5-flash, got %s", decision.SelectedModel)
	}
}

func TestRouterAdvancedModeClassifier(t *testing.T) {
	repo, err := db.NewSQLiteRepo(":memory:")
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	router := NewRouter(repo, repo)

	cfg := &db.UserConfig{
		UserID:          "user-adv",
		DailyTokenCap:   100000,
		RoutingStrategy: "advanced",
	}
	_ = repo.UpdateUserConfig(ctx, cfg)

	// Test 1: High complexity code prompt -> Upgraded to premium model
	codeDecision, err := router.EvaluateRoute(ctx, RouteRequest{
		UserID:         "user-adv",
		SessionID:      "sess-adv-1",
		Prompt:         "func HandleConcurreny(ch chan int) {\n    // Fix race condition and deadlock in goroutine channel algorithm\n}",
		RequestedModel: "gemini-1.5-flash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if codeDecision.SelectedModel != "claude-3-5-sonnet" {
		t.Errorf("expected upgrade to claude-3-5-sonnet, got %s", codeDecision.SelectedModel)
	}

	// Test 2: Low complexity casual prompt -> Downshifted to lightweight model
	simpleDecision, err := router.EvaluateRoute(ctx, RouteRequest{
		UserID:         "user-adv",
		SessionID:      "sess-adv-2",
		Prompt:         "Hi, hello! What is 2 + 2?",
		RequestedModel: "gpt-4o",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if simpleDecision.SelectedModel != "gemini-1.5-flash" {
		t.Errorf("expected downshift to gemini-1.5-flash, got %s", simpleDecision.SelectedModel)
	}
}
