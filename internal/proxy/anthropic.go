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

type AnthropicClient struct{}

type anthropicEvent struct {
	Type  string `json:"type"`
	Delta *struct {
		Text string `json:"text"`
	} `json:"delta"`
	Usage *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage"`
	Message *struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func (c *AnthropicClient) StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		mock := &MockClient{}
		return mock.StreamChat(ctx, req, chunkChan)
	}

	payload := map[string]interface{}{
		"model":      req.Model,
		"max_tokens": 1024,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": req.Prompt},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("anthropic API error (%d): %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	var inputTokens, outputTokens int64

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var evt anthropicEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		if evt.Message != nil {
			inputTokens = evt.Message.Usage.InputTokens
		}

		if evt.Type == "content_block_delta" && evt.Delta != nil && evt.Delta.Text != "" {
			chunkChan <- StreamChunk{TextDelta: evt.Delta.Text}
		}

		if evt.Usage != nil {
			outputTokens = evt.Usage.OutputTokens
		}
	}

	chunkChan <- StreamChunk{
		InputTokensConsumed: inputTokens,
		OutputTokensConsumed: outputTokens,
		IsFinal:             true,
	}

	return scanner.Err()
}
