package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type OpenAIClient struct{}

type openAIChatResponseChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

func (c *OpenAIClient) StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Fallback to mock stream if API key is not configured
		mock := &MockClient{}
		return mock.StreamChat(ctx, req, chunkChan)
	}

	payload := map[string]interface{}{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
		"stream": true,
		"stream_options": map[string]bool{
			"include_usage": true,
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai API error (%d): %s", resp.StatusCode, string(b))
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
		InputTokensConsumed: inputTokens,
		OutputTokensConsumed: outputTokens,
		IsFinal:             true,
	}

	return scanner.Err()
}
