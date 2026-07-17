package llm

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// This file adds OpenAI Responses API support without a second parser. The
// strategy is translation at both ends:
//
//   - BuildResponsesBody lowers OpenAI chat-completions messages + tools into a
//     Responses request body (item-oriented input, system prompt in
//     `instructions`, tools flattened).
//   - ResponsesToChatSSE converts the Responses `response.*` SSE event grammar
//     back into the chat-completions `data: {choices:[{delta}]}` frames every
//     existing streaming consumer already understands.
//
// So chat (its inline SSE parser) and research (the ChatComplete* helpers) share
// one implementation: they keep speaking chat-completions and only swap the wire
// body and stream reader when a server uses the Responses API.

// responsesRequest is the Responses API request body.
type responsesRequest struct {
	Model             string          `json:"model"`
	Store             bool            `json:"store"`
	Stream            bool            `json:"stream"`
	Instructions      string          `json:"instructions,omitempty"`
	Input             []any           `json:"input"`
	Tools             []responsesTool `json:"tools,omitempty"`
	ToolChoice        string          `json:"tool_choice,omitempty"`
	ParallelToolCalls bool            `json:"parallel_tool_calls,omitempty"`
	MaxOutputTokens   int             `json:"max_output_tokens,omitempty"`
	PromptCacheKey    string          `json:"prompt_cache_key,omitempty"`
	Include           []string        `json:"include,omitempty"`
}

type responsesTool struct {
	Type        string          `json:"type"` // "function"
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// BuildResponsesBody lowers chat-completions messages and tools into a Responses
// request body. messages and tools are any value that marshals to the OpenAI
// chat-completions JSON shape (e.g. a []chatMsg or []map[string]any), so callers
// need not change how they assemble them. extra may carry "max_tokens" (mapped
// to max_output_tokens); "temperature" is intentionally dropped — the Codex
// Responses endpoint rejects it.
func BuildResponsesBody(model string, messages any, tools any, extra map[string]any, stream bool) ([]byte, error) {
	msgs, err := toMaps(messages)
	if err != nil {
		return nil, fmt.Errorf("responses: lowering messages: %w", err)
	}
	instructions, input := lowerMessages(msgs)

	req := responsesRequest{
		Model:        model,
		Store:        false,
		Stream:       stream,
		Instructions: instructions,
		Input:        input,
		Include:      []string{"reasoning.encrypted_content"},
	}
	if mt, ok := intFrom(extra["max_tokens"]); ok {
		req.MaxOutputTokens = mt
	}
	if k, ok := extra["prompt_cache_key"].(string); ok {
		req.PromptCacheKey = k
	}

	toolMaps, _ := toMaps(tools)
	for _, t := range toolMaps {
		fn, _ := t["function"].(map[string]any)
		if fn == nil {
			continue
		}
		rt := responsesTool{Type: "function"}
		rt.Name, _ = fn["name"].(string)
		rt.Description, _ = fn["description"].(string)
		if params, ok := fn["parameters"]; ok {
			if raw, err := json.Marshal(params); err == nil {
				rt.Parameters = raw
			}
		}
		req.Tools = append(req.Tools, rt)
	}
	if len(req.Tools) > 0 {
		req.ToolChoice = "auto"
		req.ParallelToolCalls = true
	}
	return json.Marshal(req)
}

// lowerMessages converts chat-completions messages into Responses instructions
// (concatenated system messages) plus item-oriented input.
func lowerMessages(msgs []map[string]any) (instructions string, input []any) {
	var sys []string
	for _, m := range msgs {
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		switch role {
		case "system":
			if content != "" {
				sys = append(sys, content)
			}
		case "user":
			input = append(input, map[string]any{
				"role":    "user",
				"content": []any{map[string]any{"type": "input_text", "text": content}},
			})
		case "assistant":
			if content != "" {
				input = append(input, map[string]any{
					"role":    "assistant",
					"type":    "message",
					"status":  "completed",
					"content": []any{map[string]any{"type": "output_text", "text": content, "annotations": []any{}}},
				})
			}
			for _, tc := range toMapSlice(m["tool_calls"]) {
				fn, _ := tc["function"].(map[string]any)
				callID, _ := tc["id"].(string)
				name, _ := fn["name"].(string)
				args, _ := fn["arguments"].(string)
				input = append(input, map[string]any{
					"type":      "function_call",
					"call_id":   callID,
					"name":      name,
					"arguments": args,
				})
			}
		case "tool":
			callID, _ := m["tool_call_id"].(string)
			input = append(input, map[string]any{
				"type":    "function_call_output",
				"call_id": callID,
				"output":  content,
			})
		}
	}
	return strings.Join(sys, "\n\n"), input
}

// ResponsesToChatSSE returns a reader that converts a Responses SSE stream into
// chat-completions SSE frames. It runs the conversion in a goroutine feeding an
// io.Pipe, so the result can be handed to any consumer that already reads
// chat-completions "data:" lines (ScanSSE and the chat handler's inline parser).
func ResponsesToChatSSE(r io.Reader) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		conv := &responsesConverter{w: pw}
		err := ScanSSE(r, conv.onEvent)
		switch {
		case err != nil:
			// Reading/decoding the upstream Responses stream failed. Name it so a
			// bare transport error (e.g. "invalid argument") is traceable to the
			// Responses upstream rather than the chat-completions parser below it.
			err = fmt.Errorf("reading responses stream (saw %d event(s), %d byte(s)): %w",
				conv.events, conv.bytes, err)
		default:
			err = conv.finishIfNeeded()
		}
		pw.CloseWithError(err)
	}()
	return pr
}

// responsesConverter turns Responses events into chat-completions frames.
type responsesConverter struct {
	w             io.Writer
	toolIndex     map[int]int // Responses output_index → chat tool_calls index
	nextTool      int
	sawToolCall   bool
	finishWritten bool
	events        int // upstream events observed (diagnostics)
	bytes         int // upstream data bytes observed (diagnostics)
}

type responsesEvent struct {
	Type        string          `json:"type"`
	OutputIndex int             `json:"output_index"`
	Delta       string          `json:"delta"`
	Arguments   string          `json:"arguments"`
	Item        json.RawMessage `json:"item"`
	Response    *struct {
		Status string `json:"status"`
		Usage  *struct {
			InputTokens        int64 `json:"input_tokens"`
			OutputTokens       int64 `json:"output_tokens"`
			TotalTokens        int64 `json:"total_tokens"`
			InputTokensDetails *struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"input_tokens_details"`
		} `json:"usage"`
	} `json:"response"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *responsesConverter) onEvent(data string) error {
	c.events++
	c.bytes += len(data)
	if data == "[DONE]" {
		return nil
	}
	var ev responsesEvent
	if json.Unmarshal([]byte(data), &ev) != nil {
		return nil // tolerate keep-alives / partial noise
	}
	switch ev.Type {
	case "response.output_item.added":
		var item struct {
			Type   string `json:"type"`
			CallID string `json:"call_id"`
			Name   string `json:"name"`
		}
		if json.Unmarshal(ev.Item, &item) == nil && item.Type == "function_call" {
			c.sawToolCall = true
			if c.toolIndex == nil {
				c.toolIndex = map[int]int{}
			}
			k := c.nextTool
			c.nextTool++
			c.toolIndex[ev.OutputIndex] = k
			return c.writeChunk(map[string]any{
				"choices": []any{map[string]any{"index": 0, "delta": map[string]any{
					"tool_calls": []any{map[string]any{
						"index":    k,
						"id":       item.CallID,
						"type":     "function",
						"function": map[string]any{"name": item.Name, "arguments": ""},
					}},
				}}},
			})
		}
	case "response.output_text.delta":
		if ev.Delta != "" {
			return c.writeChunk(map[string]any{
				"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": ev.Delta}}},
			})
		}
	case "response.function_call_arguments.delta":
		if k, ok := c.toolIndex[ev.OutputIndex]; ok && ev.Delta != "" {
			return c.writeChunk(map[string]any{
				"choices": []any{map[string]any{"index": 0, "delta": map[string]any{
					"tool_calls": []any{map[string]any{
						"index":    k,
						"function": map[string]any{"arguments": ev.Delta},
					}},
				}}},
			})
		}
	case "response.completed", "response.incomplete":
		return c.finish(ev)
	case "response.failed", "error":
		msg := ev.Message
		if msg == "" {
			msg = ev.Code
		}
		return fmt.Errorf("responses: stream error: %s", msg)
	}
	return nil
}

// finish emits the terminal chat-completions frame carrying finish_reason and
// usage. Reasoning deltas are intentionally not forwarded as content.
func (c *responsesConverter) finish(ev responsesEvent) error {
	c.finishWritten = true
	reason := "stop"
	if c.sawToolCall {
		reason = "tool_calls"
	} else if ev.Type == "response.incomplete" {
		reason = "length"
	}
	frame := map[string]any{
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}},
	}
	if ev.Response != nil && ev.Response.Usage != nil {
		u := ev.Response.Usage
		usage := map[string]any{
			"prompt_tokens":     u.InputTokens,
			"completion_tokens": u.OutputTokens,
			"total_tokens":      u.TotalTokens,
		}
		if u.InputTokensDetails != nil {
			usage["prompt_tokens_details"] = map[string]any{"cached_tokens": u.InputTokensDetails.CachedTokens}
		}
		frame["usage"] = usage
	}
	return c.writeChunk(frame)
}

// finishIfNeeded emits a stop frame if the stream ended without a terminal event,
// so consumers always see a finish_reason.
func (c *responsesConverter) finishIfNeeded() error {
	if c.finishWritten {
		return nil
	}
	reason := "stop"
	if c.sawToolCall {
		reason = "tool_calls"
	}
	return c.writeChunk(map[string]any{
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": reason}},
	})
}

func (c *responsesConverter) writeChunk(frame map[string]any) error {
	b, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.w, "data: %s\n\n", b)
	return err
}

// toMaps marshals v and unmarshals it into a slice of maps, giving a
// shape-agnostic view of any chat-completions message/tool slice.
func toMaps(v any) ([]map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func toMapSlice(v any) []map[string]any {
	items, _ := v.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		if m, ok := it.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func intFrom(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}
