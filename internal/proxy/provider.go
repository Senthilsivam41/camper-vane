package proxy

import (
	"context"
	"strings"
)

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Model     string `json:"model"`
}

type StreamChunk struct {
	TextDelta           string `json:"text_delta,omitempty"`
	InputTokensConsumed int64  `json:"input_tokens_consumed,omitempty"`
	OutputTokensConsumed int64 `json:"output_tokens_consumed,omitempty"`
	IsFinal             bool   `json:"is_final,omitempty"`
	Error               error  `json:"-"`
}

type ProviderClient interface {
	StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error
}

func GetProviderClient(model string) ProviderClient {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "gpt") || strings.HasPrefix(m, "openai"):
		return &OpenAIClient{}
	case strings.HasPrefix(m, "claude") || strings.HasPrefix(m, "anthropic"):
		return &AnthropicClient{}
	case strings.HasPrefix(m, "gemini") || strings.HasPrefix(m, "google"):
		return &GeminiClient{}
	default:
		return &MockClient{}
	}
}
