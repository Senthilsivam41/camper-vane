package proxy

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type MockClient struct{}

func (c *MockClient) StreamChat(ctx context.Context, req ChatRequest, chunkChan chan<- StreamChunk) error {
	words := strings.Split(fmt.Sprintf("Responding to prompt: '%s' via negotiated model [%s]. System operation nominal.", req.Prompt, req.Model), " ")

	inputTokens := int64(len(strings.Fields(req.Prompt))) + 10
	outputTokens := int64(len(words))

	for i, w := range words {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			suffix := " "
			if i == len(words)-1 {
				suffix = ""
			}
			chunkChan <- StreamChunk{TextDelta: w + suffix}
			time.Sleep(20 * time.Millisecond)
		}
	}

	chunkChan <- StreamChunk{
		InputTokensConsumed: inputTokens,
		OutputTokens:        outputTokens,
		IsFinal:             true,
	}

	return nil
}
