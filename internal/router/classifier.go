package router

import (
	"math"
	"regexp"
	"strings"
	"unicode"

	"camper-vane/internal/db"
)

var (
	codeBlockRegex  = regexp.MustCompile("(?s)```.*?```")
	syntaxKwRegex   = regexp.MustCompile(`(?i)\b(func|def|class|interface|struct|impl|import|package|type|const|var|return|async|await|try|catch|panic|select|from|where|join|lambda|yield|mutex|rwlock)\b`)
	archKwRegex     = regexp.MustCompile(`(?i)\b(architecture|concurrency|goroutine|channel|deadlock|mutex|race condition|distributed|microservice|refactor|benchmark|optimization|algorithm|database|sqlite|postgres|memory leak|sse|oauth2|jwt|auth|security|proxy|kubernetes|throughput|latency|scalability|idempotent|event-driven|cqrs|saga)\b`)
	debugKwRegex    = regexp.MustCompile(`(?i)\b(bug|stacktrace|null pointer|segfault|timeout|reproduce|root cause|flaky|regression|heisenbug|oom|panic)\b`)
	intentHardRegex = regexp.MustCompile(`(?i)\b(implement|design|architect|migrate|diagnose|optimize|refactor|secure|benchmark|compare trade-?offs|root[- ]cause)\b`)
	intentSoftRegex = regexp.MustCompile(`(?i)\b(summarize|translate|format|rename|typo|capitalize|list|what is|define|who is)\b`)
	simpleTextRegex = regexp.MustCompile(`(?i)^(hi|hello|hey|what is|define|who is|explain simply|thanks|thank you|yes|no|\?)\b`)
	tokenSplitRegex = regexp.MustCompile(`[^\p{L}\p{N}_+#.]+`)
)

// Domain prototypes approximate a lightweight semantic centroid for engineering vs casual text.
var (
	complexPrototype = map[string]float64{
		"architecture": 1.0, "distributed": 1.0, "concurrency": 1.0, "goroutine": 0.9,
		"deadlock": 0.9, "microservice": 0.9, "refactor": 0.8, "benchmark": 0.8,
		"algorithm": 0.8, "security": 0.8, "oauth2": 0.7, "kubernetes": 0.8,
		"throughput": 0.7, "latency": 0.7, "scalability": 0.8, "database": 0.6,
		"mutex": 0.8, "channel": 0.7, "proxy": 0.6, "idempotent": 0.7,
		"implement": 0.6, "diagnose": 0.7, "optimize": 0.7, "migrate": 0.6,
	}
	simplePrototype = map[string]float64{
		"hello": 1.0, "thanks": 1.0, "please": 0.5, "what": 0.6, "define": 0.8,
		"who": 0.6, "summarize": 0.7, "translate": 0.7, "format": 0.7,
		"rename": 0.6, "typo": 0.8, "list": 0.5, "yes": 0.9, "no": 0.9,
	}
)

type ComplexityResult struct {
	Score              float64
	HasCodeBlocks      bool
	HasSyntaxKeywords  bool
	HasArchKeywords    bool
	HasDebugKeywords   bool
	IsSimpleGreeting   bool
	TotalContextTokens int
	SemanticComplex    float64
	SemanticSimple     float64
	LexicalDiversity   float64
	Method             string
	Signals            []string
}

func ClassifyComplexity(prompt string, history []db.SessionMessage) ComplexityResult {
	var combinedText strings.Builder
	for _, msg := range history {
		combinedText.WriteString(msg.Content)
		combinedText.WriteString("\n")
	}
	combinedText.WriteString(prompt)

	text := combinedText.String()
	signals := make([]string, 0, 10)

	hasCode := codeBlockRegex.MatchString(text)
	syntaxMatches := syntaxKwRegex.FindAllString(text, -1)
	archMatches := archKwRegex.FindAllString(text, -1)
	debugMatches := debugKwRegex.FindAllString(text, -1)
	hardIntent := intentHardRegex.FindAllString(prompt, -1)
	softIntent := intentSoftRegex.FindAllString(prompt, -1)
	isSimple := simpleTextRegex.MatchString(strings.TrimSpace(prompt))

	words := strings.Fields(text)
	wordCount := len(words)
	lineCount := strings.Count(text, "\n") + 1
	braceDepth := maxNesting(text, '{', '}')
	parenDepth := maxNesting(text, '(', ')')
	punctDensity := punctuationDensity(prompt)
	diversity := lexicalDiversity(text)
	bow := bagOfWords(text)
	semComplex := cosineSimilarity(bow, complexPrototype)
	semSimple := cosineSimilarity(bow, simplePrototype)
	historyBoost := historyTopicBoost(prompt, history)

	score := 0.08

	if isSimple && len(strings.Fields(prompt)) < 8 && semComplex < 0.15 {
		score = 0.04
		signals = append(signals, "simple_greeting")
	}

	// Semantic centroid blend (primary upgrade over pure keyword counting).
	score += semComplex * 0.45
	score -= semSimple * 0.25
	if semComplex > 0.25 {
		signals = append(signals, "semantic_complex_domain")
	}
	if semSimple > 0.3 && semComplex < 0.15 {
		signals = append(signals, "semantic_casual_domain")
	}

	if hasCode {
		score += 0.28
		signals = append(signals, "code_block")
	}
	if len(syntaxMatches) > 0 {
		score += math.Min(0.28, float64(len(syntaxMatches))*0.06)
		signals = append(signals, "syntax_keywords")
	}
	if len(archMatches) > 0 {
		score += math.Min(0.33, float64(len(archMatches))*0.1)
		signals = append(signals, "architecture_keywords")
	}
	if len(debugMatches) > 0 {
		score += math.Min(0.24, float64(len(debugMatches))*0.08)
		signals = append(signals, "debug_keywords")
	}
	if len(hardIntent) > 0 {
		score += 0.14
		signals = append(signals, "hard_intent")
	}
	if len(softIntent) > 0 && len(hardIntent) == 0 && !hasCode {
		score -= 0.08
		signals = append(signals, "soft_intent")
	}
	if wordCount > 100 {
		score += 0.16
		signals = append(signals, "long_context")
	} else if wordCount > 45 {
		score += 0.07
	}
	if lineCount >= 6 {
		score += 0.07
		signals = append(signals, "multi_line")
	}
	if braceDepth >= 2 || parenDepth >= 3 {
		score += 0.1
		signals = append(signals, "nested_structure")
	}
	if diversity > 0.65 && wordCount > 20 {
		score += 0.06
		signals = append(signals, "high_lexical_diversity")
	}
	if punctDensity > 0.12 && len(strings.Fields(prompt)) > 12 {
		score += 0.04
	}
	if historyBoost > 0 {
		score += historyBoost
		signals = append(signals, "history_topic_continuity")
	}

	if score < 0 {
		score = 0
	}
	if score > 1.0 {
		score = 1.0
	}

	return ComplexityResult{
		Score:              score,
		HasCodeBlocks:      hasCode,
		HasSyntaxKeywords:  len(syntaxMatches) > 0,
		HasArchKeywords:    len(archMatches) > 0,
		HasDebugKeywords:   len(debugMatches) > 0,
		IsSimpleGreeting:   isSimple,
		TotalContextTokens: wordCount,
		SemanticComplex:    semComplex,
		SemanticSimple:     semSimple,
		LexicalDiversity:   diversity,
		Method:             "semantic_heuristic_v2",
		Signals:            signals,
	}
}

func historyTopicBoost(prompt string, history []db.SessionMessage) float64 {
	if len(history) == 0 {
		return 0
	}
	promptBow := bagOfWords(prompt)
	var hist strings.Builder
	for _, msg := range history {
		hist.WriteString(msg.Content)
		hist.WriteByte(' ')
	}
	overlap := cosineSimilarity(promptBow, bagOfWords(hist.String()))
	histComplex := cosineSimilarity(bagOfWords(hist.String()), complexPrototype)
	if histComplex > 0.2 && overlap > 0.1 {
		return 0.12
	}
	if histComplex > 0.25 {
		return 0.08
	}
	return 0
}

func bagOfWords(text string) map[string]float64 {
	parts := tokenSplitRegex.Split(strings.ToLower(text), -1)
	out := make(map[string]float64, len(parts))
	for _, p := range parts {
		if p == "" || len(p) < 2 {
			continue
		}
		out[p]++
	}
	return out
}

func cosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot, na, nb float64
	for k, va := range a {
		na += va * va
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	for _, vb := range b {
		nb += vb * vb
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func lexicalDiversity(text string) float64 {
	parts := tokenSplitRegex.Split(strings.ToLower(text), -1)
	if len(parts) == 0 {
		return 0
	}
	uniq := map[string]struct{}{}
	total := 0
	for _, p := range parts {
		if p == "" {
			continue
		}
		total++
		uniq[p] = struct{}{}
	}
	if total == 0 {
		return 0
	}
	return float64(len(uniq)) / float64(total)
}

func maxNesting(s string, open, close rune) int {
	depth, maxDepth := 0, 0
	for _, r := range s {
		if r == open {
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		} else if r == close && depth > 0 {
			depth--
		}
	}
	return maxDepth
}

func punctuationDensity(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	punct := 0
	runes := []rune(s)
	for _, r := range runes {
		if unicode.IsPunct(r) {
			punct++
		}
	}
	return float64(punct) / float64(len(runes))
}
