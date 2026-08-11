package router

import (
	"fmt"
	"strings"
)

// Approximate USD cost per 1K tokens (blended input/output) for routing deltas.
var modelCostPer1K = map[string]float64{
	"gemini-1.5-flash":  0.00015,
	"gpt-4o-mini":       0.0003,
	"sonar":             0.001,
	"sonar-pro":         0.003,
	"gpt-4o":            0.005,
	"claude-3-5-sonnet": 0.009,
	"claude-3-opus":     0.03,
}

const baselineModel = "gemini-1.5-flash"
const assumedPromptTokens = 800.0 // heuristic for pre-stream cost delta display

func CostPer1K(model string) float64 {
	if c, ok := modelCostPer1K[strings.ToLower(model)]; ok {
		return c
	}
	return 0.002
}

// FormatCostDelta compares selected model vs baseline ultra-low-cost model.
func FormatCostDelta(selectedModel string) string {
	delta := (CostPer1K(selectedModel) - CostPer1K(baselineModel)) * (assumedPromptTokens / 1000.0)
	if delta > -0.00005 && delta < 0.00005 {
		return "$0.0000"
	}
	if delta >= 0 {
		return fmt.Sprintf("+$%.4f", delta)
	}
	return fmt.Sprintf("-$%.4f", -delta)
}

// PickPreferredModel chooses from preferred list: premium=true picks first high-tier, else first low-tier.
func PickPreferredModel(preferred []string, premium bool, fallback string) string {
	if len(preferred) == 0 {
		return fallback
	}
	lowCost := map[string]bool{
		"gemini-1.5-flash": true,
		"gpt-4o-mini":      true,
		"sonar":            true,
	}
	if premium {
		for _, m := range preferred {
			if !lowCost[strings.ToLower(m)] {
				return m
			}
		}
		return preferred[0]
	}
	for _, m := range preferred {
		if lowCost[strings.ToLower(m)] {
			return m
		}
	}
	return preferred[len(preferred)-1]
}
