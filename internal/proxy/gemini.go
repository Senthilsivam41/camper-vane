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

type GeminiClient struct{}

type geminiResponseChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (c *GeminiClient) StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		mock := &MockClient{}
		return mock.StreamChat(ctx, req, chunkChan)
	}

	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", modelName, apiKey)

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": req.Prompt},
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gemini API error (%d): %s", resp.StatusCode, string(b))
	}

	scanner := bufio.NewScanner(resp.Body)
	var inputTokens, outputTokens int64

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk geminiResponseChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Candidates) > 0 && len(chunk.Candidates[0].Content.Parts) > 0 {
			text := chunk.Candidates[0].Content.Parts[0].Text
			if text != "" {
				chunkChan <- StreamChunk{TextDelta: text}
			}
		}

		if chunk.UsageMetadata != nil {
			inputTokens = chunk.UsageMetadata.PromptTokenCount
			outputTokens = chunk.UsageMetadata.CandidatesTokenCount
		}
	}

	chunkChan <- StreamChunk{
		InputTokensConsumed: inputTokens,
		OutputTokensConsumed: outputTokens,
		IsFinal:             true,
	}

	return scanner.Err()
}
