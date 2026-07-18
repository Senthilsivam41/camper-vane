package router

import (
	"context"
	"fmt"
	"time"

	"camper-vane/internal/db"
)

type RouteRequest struct {
	UserID        string
	SessionID     string
	Prompt        string
	RequestedModel string
}

type RoutingDecision struct {
	SelectedModel      string  `json:"selected_model"`
	RoutingRationale   string  `json:"routing_rationale"`
	EstimatedCostDelta string  `json:"estimated_cost_delta"`
	BudgetThrottled    bool    `json:"budget_throttled"`
	ComplexityScore    float64 `json:"complexity_score"`
}

type Router struct {
	userRepo    db.UserRepository
	sessionRepo db.SessionRepository
}

func NewRouter(userRepo db.UserRepository, sessionRepo db.SessionRepository) *Router {
	return &Router{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

func (r *Router) EvaluateRoute(ctx context.Context, req RouteRequest) (*RoutingDecision, error) {
	// 1. Fetch user config and daily usage
	userCfg, err := r.userRepo.GetUserConfig(ctx, req.UserID)
	if err != nil {
		userCfg = &db.UserConfig{DailyTokenCap: 50000, RoutingStrategy: "simple"}
	}

	today := time.Now()
	dailyUsage, err := r.userRepo.GetDailyUsage(ctx, req.UserID, today)
	if err != nil {
		dailyUsage = 0
	}

	// 2. Check Tier 1: Simple Mode Volumetric Throttle (>= 85% cap)
	var utilization float64
	if userCfg.DailyTokenCap > 0 {
		utilization = (float64(dailyUsage) / float64(userCfg.DailyTokenCap)) * 100.0
	}

	if utilization >= 85.0 || userCfg.RoutingStrategy == "simple" {
		if utilization >= 85.0 {
			return &RoutingDecision{
				SelectedModel:      "gemini-1.5-flash",
				RoutingRationale:   fmt.Sprintf("Volumetric budget warning: Current utilization is %.1f%% of daily cap (%d/%d tokens). Forced ultra-low-cost model execution.", utilization, dailyUsage, userCfg.DailyTokenCap),
				EstimatedCostDelta: "-$0.0045",
				BudgetThrottled:    true,
				ComplexityScore:    0.1,
			}, nil
		}
	}

	// 3. Tier 2: Advanced Mode Context Hydration & Semantic Classification
	history, err := r.sessionRepo.GetSessionHistory(ctx, req.SessionID, 5)
	if err != nil {
		history = nil
	}

	cls := ClassifyComplexity(req.Prompt, history)

	selectedModel := req.RequestedModel
	var costDelta string
	var rationale string

	if cls.Score >= 0.5 {
		if selectedModel == "" || selectedModel == "gemini-1.5-flash" || selectedModel == "gpt-4o-mini" {
			selectedModel = "claude-3-5-sonnet"
		}
		costDelta = "+$0.0035"
		rationale = fmt.Sprintf("Advanced Mode: Query complexity score (%.2f) signals architectural/code logic. Negotiated premium engine [%s].", cls.Score, selectedModel)
	} else {
		selectedModel = "gemini-1.5-flash"
		costDelta = "-$0.0021"
		rationale = fmt.Sprintf("Advanced Mode: Query complexity score (%.2f) signals standard formatting/syntax. Transparently routed to lightweight tier [%s] for cost savings.", cls.Score, selectedModel)
	}

	return &RoutingDecision{
		SelectedModel:      selectedModel,
		RoutingRationale:   rationale,
		EstimatedCostDelta: costDelta,
		BudgetThrottled:    false,
		ComplexityScore:    cls.Score,
	}, nil
}
