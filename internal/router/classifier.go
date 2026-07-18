package router

import (
	"regexp"
	"strings"

	"camper-vane/internal/db"
)

var (
	codeBlockRegex  = regexp.MustCompile("(?s)```.*?```")
	syntaxKwRegex   = regexp.MustCompile(`\b(func|def|class|interface|struct|impl|import|package|type|const|var|return|async|await|try|catch|panic)\b`)
	archKwRegex     = regexp.MustCompile(`(?i)\b(architecture|concurrency|goroutine|channel|deadlock|mutex|race condition|distributed|microservice|refactor|benchmark|optimization|algorithm|database|sqlite|postgres|memory leak|sse|oauth2|jwt|auth|security|proxy)\b`)
	simpleTextRegex = regexp.MustCompile(`(?i)^(hi|hello|hey|what is|define|who is|explain simply|thanks|thank you|yes|no|\?)\b`)
)

type ComplexityResult struct {
	Score              float64
	HasCodeBlocks      bool
	HasSyntaxKeywords  bool
	HasArchKeywords    bool
	IsSimpleGreeting   bool
	TotalContextTokens int
}

func ClassifyComplexity(prompt string, history []db.SessionMessage) ComplexityResult {
	var combinedText strings.Builder
	for _, msg := range history {
		combinedText.WriteString(msg.Content)
		combinedText.WriteString("\n")
	}
	combinedText.WriteString(prompt)

	text := combinedText.String()

	hasCode := codeBlockRegex.MatchString(text)
	syntaxMatches := syntaxKwRegex.FindAllString(text, -1)
	archMatches := archKwRegex.FindAllString(text, -1)
	isSimple := simpleTextRegex.MatchString(strings.TrimSpace(prompt))

	wordCount := len(strings.Fields(text))

	score := 0.1 // Base score

	if isSimple && len(strings.Fields(prompt)) < 8 {
		score = 0.05
	}

	if hasCode {
		score += 0.4
	}

	score += float64(len(syntaxMatches)) * 0.08
	score += float64(len(archMatches)) * 0.12

	if wordCount > 80 {
		score += 0.2
	} else if wordCount > 40 {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return ComplexityResult{
		Score:              score,
		HasCodeBlocks:      hasCode,
		HasSyntaxKeywords:  len(syntaxMatches) > 0,
		HasArchKeywords:    len(archMatches) > 0,
		IsSimpleGreeting:   isSimple,
		TotalContextTokens: wordCount,
	}
}
