// Package llm holds shared helpers for talking to OpenAI-compatible model
// servers: the SSE streaming scanner used by every streaming handler, and a
// non-streaming chat-completion call used by the title worker, the web-page
// summariser, and the research engine.
package llm

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

// sseDataPrefix is the field prefix on each Server-Sent Events data line.
const sseDataPrefix = "data: "

// maxResponseBytes caps how much of a non-streaming response body we read, so a
// misbehaving or malicious model server cannot exhaust memory.
const maxResponseBytes = 20 * 1024 * 1024

// Message is a single OpenAI-style chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage is the billing metadata returned by OpenAI-compatible providers.
// Cost is provider-reported rather than recomputed locally, so it includes
// routing, prompt-cache reads/writes, reasoning, and other provider-specific
// pricing rules. A nil Cost means the provider did not report a price; zero is
// a known free call.
type Usage struct {
	PromptTokens     int      `json:"prompt_tokens"`
	CompletionTokens int      `json:"completion_tokens"`
	TotalTokens      int      `json:"total_tokens"`
	Cost             *float64 `json:"cost"`
	// PromptTokensDetails carries the cached-prompt-token count when the provider
	// reports it (OpenAI/Responses prompt caching). Read via CachedTokens().
	PromptTokensDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details,omitempty"`
}

// CachedTokens returns the number of prompt tokens served from cache, or 0.
func (u *Usage) CachedTokens() int {
	if u == nil || u.PromptTokensDetails == nil {
		return 0
	}
	return u.PromptTokensDetails.CachedTokens
}

type Completion struct {
	Content string
	Usage   *Usage
	// FinishReason is the provider's stop reason for the first choice, e.g.
	// "stop" (natural end), "length" (hit max_tokens — output truncated), or
	// "tool_calls". Empty when the provider did not report one.
	FinishReason string
	// ToolCalls holds any function calls parsed from the stream (provider seam).
	ToolCalls []ToolCall
	// Thinking is accumulated reasoning/thinking text. Reserved: persisted for
	// display but never sent back to the model. Not yet populated.
	Thinking string
}

func (c Completion) UsageCost() *float64 {
	if c.Usage == nil {
		return nil
	}
	return c.Usage.Cost
}

// ScanSSE reads an OpenAI-style Server-Sent Events stream from body, invoking
// onData with the payload of each "data: " line until it reaches "[DONE]" or
// the stream ends. Non-data lines (comments, blanks) are skipped. If onData
// returns an error the scan stops and that error is returned; otherwise any
// read error from the underlying scanner is returned (nil on clean EOF).
//
// The 1 MiB buffer matches the largest single SSE frame a model server is
// expected to emit; without it bufio.Scanner errors on long tool-call frames.
func ScanSSE(body io.Reader, onData func(data string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		data := line[len(sseDataPrefix):]
		if data == "[DONE]" {
			return nil
		}
		if err := onData(data); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		// Include the concrete error type: transport failures surface here as
		// opaque strings ("invalid argument", "unexpected EOF") and the type is
		// often the only thing that distinguishes them.
		return fmt.Errorf("sse scan: %w (%T)", err, err)
	}
	return nil
}

// ChatComplete makes a non-streaming POST to a /chat/completions endpoint and
// returns the (untrimmed) content of the first choice. messages is marshalled
// as-is, so callers may pass any slice whose elements carry the OpenAI message
// JSON shape (e.g. []Message). extra supplies optional payload fields such as
// "temperature" and "max_tokens"; "model", "messages", and "stream" are set by
// this function and must not be passed in extra.
func ChatComplete(ctx context.Context, client *http.Client, chatURL, apiKey, model string, messages any, extra map[string]any) (string, error) {
	result, err := ChatCompleteWithUsage(ctx, client, chatURL, apiKey, model, messages, extra)
	return result.Content, err
}

// ChatCompleteWithUsage is ChatComplete with provider-reported usage and cost.
func ChatCompleteWithUsage(ctx context.Context, client *http.Client, chatURL, apiKey, model string, messages any, extra map[string]any) (Completion, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
	}
	for k, v := range extra {
		body[k] = v
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Completion{}, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return Completion{}, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Completion{}, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, respBody)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return Completion{}, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return Completion{}, fmt.Errorf("no choices in response")
	}
	return Completion{
		Content:      result.Choices[0].Message.Content,
		Usage:        result.Usage,
		FinishReason: result.Choices[0].FinishReason,
	}, nil
}

// ChatCompleteStream makes a streaming chat-completion request. It returns the
// complete generated content and invokes onDelta for each non-empty text delta.
func ChatCompleteStream(ctx context.Context, client *http.Client, chatURL, apiKey, model string, messages any, extra map[string]any, onDelta func(string)) (string, error) {
	result, err := ChatCompleteStreamWithUsage(ctx, client, chatURL, apiKey, model, messages, extra, onDelta)
	return result.Content, err
}

// ChatCompleteStreamWithUsage captures the usage object sent in the final SSE
// chunk. OpenRouter sends that chunk with an empty choices array.
func ChatCompleteStreamWithUsage(ctx context.Context, client *http.Client, chatURL, apiKey, model string, messages any, extra map[string]any, onDelta func(string)) (Completion, error) {
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}
	for k, v := range extra {
		body[k] = v
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", chatURL, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Completion{}, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Completion{}, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, respBody)
	}
	return readChatCompletionsStream(resp.Body, onDelta)
}

// readChatCompletionsStream parses a chat-completions SSE body, accumulating the
// text content and capturing usage and finish reason. It is shared by the direct
// chat-completions path and the Responses path (whose stream is first converted
// to chat-completions frames), so both use one parser.
func readChatCompletionsStream(body io.Reader, onDelta func(string)) (Completion, error) {
	var full strings.Builder
	var usage *Usage
	var finishReason string
	err := ScanSSE(body, func(data string) error {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage *Usage `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return nil
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
		if len(chunk.Choices) == 0 {
			return nil
		}
		if chunk.Choices[0].FinishReason != "" {
			finishReason = chunk.Choices[0].FinishReason
		}
		delta := chunk.Choices[0].Delta.Content
		if delta != "" {
			full.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		}
		return nil
	})
	if err != nil && full.Len() == 0 {
		return Completion{}, fmt.Errorf("stream read failed: %w", err)
	}
	return Completion{Content: full.String(), Usage: usage, FinishReason: finishReason}, nil
}

// ResponsesStreamWithUsage runs a streaming request against an OpenAI Responses
// endpoint, translating the request from chat-completions messages and the
// response stream back to chat-completions frames so the result matches
// ChatCompleteStreamWithUsage. accountID, when set, is sent as the
// chatgpt-account-id header the ChatGPT-subscription backend requires.
func ResponsesStreamWithUsage(ctx context.Context, client *http.Client, url, apiKey, accountID, model string, messages any, tools any, extra map[string]any, onDelta func(string)) (Completion, error) {
	payload, err := BuildResponsesBody(model, messages, tools, extra, true)
	if err != nil {
		return Completion{}, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return Completion{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	if accountID != "" {
		req.Header.Set("chatgpt-account-id", accountID)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Completion{}, fmt.Errorf("model request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return Completion{}, fmt.Errorf("model server returned %d: %.300s", resp.StatusCode, respBody)
	}
	return readChatCompletionsStream(ResponsesToChatSSE(resp.Body), onDelta)
}
