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
	"errors"
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

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("model server returned %d: %.300s", e.StatusCode, e.Body)
}

// ErrorHTTPStatus returns the provider HTTP status carried by err, or zero for
// transport, timeout, decoding, and other non-HTTP failures.
func ErrorHTTPStatus(err error) int {
	var statusErr *HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode
	}
	return 0
}

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
}

type Completion struct {
	Content      string
	FinishReason string
	Usage        *Usage
	RawUsage     json.RawMessage
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
	return scanner.Err()
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
		return Completion{}, &HTTPStatusError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return Completion{}, fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 {
		return Completion{}, fmt.Errorf("no choices in response")
	}
	var usage *Usage
	if len(result.Usage) > 0 && string(result.Usage) != "null" {
		var parsed Usage
		if json.Unmarshal(result.Usage, &parsed) == nil {
			usage = &parsed
		}
	}
	return Completion{Content: result.Choices[0].Message.Content, FinishReason: result.Choices[0].FinishReason, Usage: usage, RawUsage: result.Usage}, nil
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
		return Completion{}, &HTTPStatusError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}
	var full strings.Builder
	var usage *Usage
	var rawUsage json.RawMessage
	var finishReason string
	err = ScanSSE(resp.Body, func(data string) error {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Usage json.RawMessage `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return nil
		}
		if len(chunk.Usage) > 0 && string(chunk.Usage) != "null" {
			rawUsage = append(rawUsage[:0], chunk.Usage...)
			var parsed Usage
			if json.Unmarshal(chunk.Usage, &parsed) == nil {
				usage = &parsed
			}
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
	completion := Completion{Content: full.String(), FinishReason: finishReason, Usage: usage, RawUsage: rawUsage}
	if err != nil {
		return completion, fmt.Errorf("stream read failed: %w", err)
	}
	return completion, nil
}
