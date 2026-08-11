package router

import (
	"testing"

	"camper-vane/internal/db"
)

func TestClassifyComplexitySignals(t *testing.T) {
	simple := ClassifyComplexity("Hi, hello!", nil)
	if simple.Score >= 0.5 {
		t.Fatalf("expected simple prompt low score, got %.2f", simple.Score)
	}
	if simple.Method != "semantic_heuristic_v2" {
		t.Fatalf("expected method semantic_heuristic_v2, got %s", simple.Method)
	}

	complex := ClassifyComplexity("```go\nfunc Handle(ch chan int) {\n  // fix deadlock race condition in microservice architecture\n}\n```", nil)
	if complex.Score < 0.5 {
		t.Fatalf("expected complex prompt high score, got %.2f signals=%v", complex.Score, complex.Signals)
	}
	if !complex.HasCodeBlocks || !complex.HasArchKeywords {
		t.Fatalf("expected code+arch signals, got %+v", complex)
	}
	if complex.SemanticComplex <= complex.SemanticSimple {
		t.Fatalf("expected semantic complex > simple, got complex=%.3f simple=%.3f", complex.SemanticComplex, complex.SemanticSimple)
	}

	withHistory := ClassifyComplexity("continue", []db.SessionMessage{
		{Role: "user", Content: "debug null pointer segfault regression in distributed proxy"},
	})
	if withHistory.Score < 0.4 {
		t.Fatalf("expected history to raise score, got %.2f", withHistory.Score)
	}
}

func TestClassifyComplexityTable(t *testing.T) {
	cases := []struct {
		name      string
		prompt    string
		history   []db.SessionMessage
		wantHigh  bool // score >= 0.5
		wantLow   bool // score < 0.35
	}{
		{
			name:     "greeting",
			prompt:   "Thanks!",
			wantLow:  true,
		},
		{
			name:     "definition",
			prompt:   "What is a linked list?",
			wantLow:  true,
		},
		{
			name:     "soft format",
			prompt:   "Please format this JSON and rename the keys",
			wantLow:  true,
		},
		{
			name:     "hard design",
			prompt:   "Design a distributed event-driven microservice architecture with idempotent consumers and CQRS",
			wantHigh: true,
		},
		{
			name:     "debug production",
			prompt:   "Diagnose a flaky race condition deadlock in our goroutine channel pipeline under load",
			wantHigh: true,
		},
		{
			name:   "history continuity",
			prompt: "how should we migrate that?",
			history: []db.SessionMessage{
				{Role: "user", Content: "We need to refactor the kubernetes authentication proxy for scalability and security"},
				{Role: "assistant", Content: "Consider token rotation and horizontal pods"},
			},
			wantHigh: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyComplexity(tc.prompt, tc.history)
			if tc.wantHigh && got.Score < 0.5 {
				t.Fatalf("want high complexity, got %.2f signals=%v semC=%.2f", got.Score, got.Signals, got.SemanticComplex)
			}
			if tc.wantLow && got.Score >= 0.35 {
				t.Fatalf("want low complexity, got %.2f signals=%v semS=%.2f", got.Score, got.Signals, got.SemanticSimple)
			}
		})
	}
}

func TestCosineAndDiversityHelpers(t *testing.T) {
	a := bagOfWords("distributed architecture concurrency")
	b := bagOfWords("distributed architecture scalability")
	sim := cosineSimilarity(a, b)
	if sim <= 0 {
		t.Fatalf("expected positive cosine, got %f", sim)
	}
	div := lexicalDiversity("one two two three three three")
	if div <= 0 || div >= 1 {
		t.Fatalf("unexpected diversity %f", div)
	}
}

func TestFormatCostDeltaAndPickPreferred(t *testing.T) {
	delta := FormatCostDelta("claude-3-5-sonnet")
	if delta[0] != '+' {
		t.Fatalf("expected positive delta for premium model, got %s", delta)
	}
	cheap := FormatCostDelta("gemini-1.5-flash")
	if cheap != "$0.0000" {
		t.Fatalf("expected zero delta vs baseline, got %s", cheap)
	}

	prefs := []string{"gpt-4o-mini", "claude-3-5-sonnet", "sonar"}
	if got := PickPreferredModel(prefs, true, "claude-3-5-sonnet"); got != "claude-3-5-sonnet" {
		t.Fatalf("premium pick: got %s", got)
	}
	if got := PickPreferredModel(prefs, false, "gemini-1.5-flash"); got != "gpt-4o-mini" {
		t.Fatalf("low-cost pick: got %s", got)
	}
}
