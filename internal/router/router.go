package router

import (
	"context"
	"fmt"
	"time"

	"camper-vane/internal/db"
)

type RouteRequest struct {
	UserID         string
	SessionID      string
	Prompt         string
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
	userCfg, err := r.userRepo.GetUserConfig(ctx, req.UserID)
	if err != nil {
		userCfg = &db.UserConfig{
			DailyTokenCap:   50000,
			RoutingStrategy: "simple",
			PreferredModels: []string{baselineModel, "gpt-4o-mini"},
		}
	}

	since := time.Now().Add(-24 * time.Hour)
	usage, err := r.userRepo.GetUsageSince(ctx, req.UserID, since)
	if err != nil {
		// Fallback to calendar-day metric if rolling window unavailable.
		usage, _ = r.userRepo.GetDailyUsage(ctx, req.UserID, time.Now())
	}

	var utilization float64
	if userCfg.DailyTokenCap > 0 {
		utilization = (float64(usage) / float64(userCfg.DailyTokenCap)) * 100.0
	}

	// Tier 1: volumetric throttle across sliding 24h window.
	if utilization >= 85.0 {
		model := PickPreferredModel(userCfg.PreferredModels, false, baselineModel)
		return &RoutingDecision{
			SelectedModel:      model,
			RoutingRationale:   fmt.Sprintf("Volumetric budget warning: trailing-24h utilization is %.1f%% of cap (%d/%d tokens). Forced ultra-low-cost model execution.", utilization, usage, userCfg.DailyTokenCap),
			EstimatedCostDelta: FormatCostDelta(model),
			BudgetThrottled:    true,
			ComplexityScore:    0.1,
		}, nil
	}

	// Simple mode (non-throttled): stick to cheapest preferred model.
	if userCfg.RoutingStrategy == "simple" {
		model := req.RequestedModel
		if model == "" {
			model = PickPreferredModel(userCfg.PreferredModels, false, baselineModel)
		}
		return &RoutingDecision{
			SelectedModel:      model,
			RoutingRationale:   fmt.Sprintf("Simple Mode: trailing-24h usage %.1f%% of cap. Routing to preferred low-cost model [%s].", utilization, model),
			EstimatedCostDelta: FormatCostDelta(model),
			BudgetThrottled:    false,
			ComplexityScore:    0.15,
		}, nil
	}

	// Tier 2: Advanced Mode — context hydration + richer classification.
	history, err := r.sessionRepo.GetSessionHistory(ctx, req.SessionID, 5)
	if err != nil {
		history = nil
	}

	cls := ClassifyComplexity(req.Prompt, history)
	premium := cls.Score >= 0.5

	selectedModel := req.RequestedModel
	if selectedModel == "" {
		selectedModel = PickPreferredModel(userCfg.PreferredModels, premium, baselineModel)
	} else if premium {
		// Upgrade away from lightweight requested models when complexity is high.
		low := map[string]bool{"gemini-1.5-flash": true, "gpt-4o-mini": true, "sonar": true}
		if low[selectedModel] {
			selectedModel = PickPreferredModel(userCfg.PreferredModels, true, "claude-3-5-sonnet")
		}
	} else {
		selectedModel = PickPreferredModel(userCfg.PreferredModels, false, baselineModel)
	}

	var rationale string
	if premium {
		rationale = fmt.Sprintf("Advanced Mode (%s): complexity score %.2f (signals: %v; semantic=%.2f). Negotiated premium engine [%s].", cls.Method, cls.Score, cls.Signals, cls.SemanticComplex, selectedModel)
	} else {
		rationale = fmt.Sprintf("Advanced Mode (%s): complexity score %.2f (signals: %v; semantic=%.2f). Routed to lightweight tier [%s] for cost savings.", cls.Method, cls.Score, cls.Signals, cls.SemanticComplex, selectedModel)
	}

	return &RoutingDecision{
		SelectedModel:      selectedModel,
		RoutingRationale:   rationale,
		EstimatedCostDelta: FormatCostDelta(selectedModel),
		BudgetThrottled:    false,
		ComplexityScore:    cls.Score,
	}, nil
}
