package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// PerplexityClient streams via Perplexity's OpenAI-compatible chat completions API.
type PerplexityClient struct{}

func (c *PerplexityClient) StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error {
	apiKey, err := ResolveAPIKey("PERPLEXITY_API_KEY")
	if err != nil {
		return err
	}
	if apiKey == "" {
		mock := &MockClient{}
		return mock.StreamChat(ctx, req, chunkChan)
	}

	model := req.Model
	if model == "" || strings.HasPrefix(strings.ToLower(model), "perplexity") {
		model = "sonar"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"stream": true,
	}

	bodyBytes, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.perplexity.ai/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("perplexity API error (%d): %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	var inputTokens, outputTokens int64

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIChatResponseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			chunkChan <- StreamChunk{TextDelta: chunk.Choices[0].Delta.Content}
		}

		if chunk.Usage != nil {
			inputTokens = chunk.Usage.PromptTokens
			outputTokens = chunk.Usage.CompletionTokens
		}
	}

	chunkChan <- StreamChunk{
		InputTokensConsumed:  inputTokens,
		OutputTokensConsumed: outputTokens,
		IsFinal:              true,
	}

	return scanner.Err()
}
