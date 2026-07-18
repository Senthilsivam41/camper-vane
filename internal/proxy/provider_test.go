package proxy

import (
	"context"
	"testing"
)

func TestProviderFactory(t *testing.T) {
	if _, ok := GetProviderClient("gpt-4o").(*OpenAIClient); !ok {
		t.Errorf("expected OpenAIClient for gpt-4o")
	}
	if _, ok := GetProviderClient("claude-3-5-sonnet").(*AnthropicClient); !ok {
		t.Errorf("expected AnthropicClient for claude-3-5-sonnet")
	}
	if _, ok := GetProviderClient("gemini-1.5-flash").(*GeminiClient); !ok {
		t.Errorf("expected GeminiClient for gemini-1.5-flash")
	}
	if _, ok := GetProviderClient("unknown-model").(*MockClient); !ok {
		t.Errorf("expected MockClient for unknown model")
	}
}

func TestMockClientStream(t *testing.T) {
	client := &MockClient{}
	ctx := context.Background()
	req := ChatRequest{
		SessionID: "sess-1",
		Prompt:    "Hello world",
		Model:     "gemini-1.5-flash",
	}

	chunkChan := make(chan StreamChunk, 50)
	go func() {
		defer close(chunkChan)
		if err := client.StreamChat(ctx, req, chunkChan); err != nil {
			t.Errorf("unexpected error in StreamChat: %v", err)
		}
	}()

	var receivedText string
	var finalChunk *StreamChunk

	for c := range chunkChan {
		if c.IsFinal {
			finalChunk = &c
		} else {
			receivedText += c.TextDelta
		}
	}

	if receivedText == "" {
		t.Errorf("expected non-empty received text delta")
	}

	if finalChunk == nil {
		t.Fatalf("expected final chunk with tokens metadata")
	}

	if finalChunk.InputTokensConsumed <= 0 || finalChunk.OutputTokens <= 0 {
		t.Errorf("expected positive token metrics, got input=%d output=%d", finalChunk.InputTokensConsumed, finalChunk.OutputTokens)
	}
}
