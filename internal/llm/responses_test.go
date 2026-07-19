package llm

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestBuildResponsesBody(t *testing.T) {
	messages := []map[string]any{
		{"role": "system", "content": "Be terse."},
		{"role": "user", "content": "Hi"},
		{"role": "assistant", "content": "prior", "tool_calls": []any{
			map[string]any{"id": "call_1", "type": "function", "function": map[string]any{"name": "get_time", "arguments": "{}"}},
		}},
		{"role": "tool", "tool_call_id": "call_1", "content": "12:00"},
	}
	tools := []map[string]any{
		{"type": "function", "function": map[string]any{"name": "get_time", "description": "now", "parameters": map[string]any{"type": "object"}}},
	}
	raw, err := BuildResponsesBody("gpt-x", messages, tools, map[string]any{"max_tokens": 500, "temperature": 0.7}, true)
	if err != nil {
		t.Fatalf("BuildResponsesBody: %v", err)
	}
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}
	if req["instructions"] != "Be terse." {
		t.Errorf("instructions = %v", req["instructions"])
	}
	if _, ok := req["temperature"]; ok {
		t.Error("temperature must be dropped for the Responses endpoint")
	}
	if _, ok := req["max_output_tokens"]; ok {
		t.Error("max_output_tokens must be dropped for the Responses endpoint")
	}
	if req["store"] != false || req["stream"] != true {
		t.Errorf("store/stream = %v/%v", req["store"], req["stream"])
	}
	input := req["input"].([]any)
	// user message, assistant message, function_call, function_call_output
	if len(input) != 4 {
		t.Fatalf("expected 4 input items, got %d: %v", len(input), input)
	}
	fc := input[2].(map[string]any)
	if fc["type"] != "function_call" || fc["call_id"] != "call_1" || fc["name"] != "get_time" {
		t.Errorf("function_call item wrong: %v", fc)
	}
	fco := input[3].(map[string]any)
	if fco["type"] != "function_call_output" || fco["call_id"] != "call_1" || fco["output"] != "12:00" {
		t.Errorf("function_call_output item wrong: %v", fco)
	}
	toolsOut := req["tools"].([]any)
	if len(toolsOut) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(toolsOut))
	}
	tool := toolsOut[0].(map[string]any)
	if tool["type"] != "function" || tool["name"] != "get_time" {
		t.Errorf("tool flattening wrong: %v", tool)
	}
	if req["tool_choice"] != "auto" {
		t.Errorf("tool_choice = %v", req["tool_choice"])
	}
}

// collectFrames reads chat-completions SSE frames from r into decoded maps.
func collectFrames(t *testing.T, r io.Reader) []map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read converted stream: %v", err)
	}
	var frames []map[string]any
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var f map[string]any
		if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &f) == nil {
			frames = append(frames, f)
		}
	}
	return frames
}

func sse(events ...string) string {
	var b strings.Builder
	for _, e := range events {
		b.WriteString("data: ")
		b.WriteString(e)
		b.WriteString("\n\n")
	}
	return b.String()
}

func TestResponsesToChatSSETextAndUsage(t *testing.T) {
	in := sse(
		`{"type":"response.created","response":{"id":"r1"}}`,
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"message"}}`,
		`{"type":"response.output_text.delta","output_index":0,"delta":"Hello"}`,
		`{"type":"response.output_text.delta","output_index":0,"delta":" world"}`,
		`{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":10,"output_tokens":3,"total_tokens":13}}}`,
	)
	comp, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if comp.Content != "Hello world" {
		t.Errorf("content = %q", comp.Content)
	}
	if comp.FinishReason != "stop" {
		t.Errorf("finish = %q", comp.FinishReason)
	}
	if comp.Usage == nil || comp.Usage.PromptTokens != 10 || comp.Usage.CompletionTokens != 3 {
		t.Errorf("usage = %+v", comp.Usage)
	}
}

func TestResponsesToChatSSEToolCalls(t *testing.T) {
	in := sse(
		`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice"}}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"sides\":"}`,
		`{"type":"response.function_call_arguments.delta","output_index":0,"delta":"6}"}`,
		`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"roll_dice","arguments":"{\"sides\":6}"}}`,
		`{"type":"response.completed","response":{"status":"completed"}}`,
	)
	frames := collectFrames(t, ResponsesToChatSSE(strings.NewReader(in)))

	// Reassemble tool call from frames the way the chat handler does.
	var id, name, args, finish string
	for _, f := range frames {
		choices, _ := f["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		ch := choices[0].(map[string]any)
		if fr, ok := ch["finish_reason"].(string); ok {
			finish = fr
		}
		delta, _ := ch["delta"].(map[string]any)
		tcs, _ := delta["tool_calls"].([]any)
		for _, tcAny := range tcs {
			tc := tcAny.(map[string]any)
			if v, ok := tc["id"].(string); ok && v != "" {
				id = v
			}
			fn, _ := tc["function"].(map[string]any)
			if v, ok := fn["name"].(string); ok && v != "" {
				name = v
			}
			if v, ok := fn["arguments"].(string); ok {
				args += v
			}
		}
	}
	if id != "c1" || name != "roll_dice" || args != `{"sides":6}` {
		t.Errorf("reassembled tool call wrong: id=%q name=%q args=%q", id, name, args)
	}
	if finish != "tool_calls" {
		t.Errorf("finish = %q, want tool_calls", finish)
	}
}

func TestResponsesToChatSSEError(t *testing.T) {
	in := sse(`{"type":"response.failed","response":{"status":"failed"},"message":"boom"}`)
	_, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err == nil {
		t.Fatal("expected error from failed response")
	}
}

// The Codex backend nests the reason under a top-level "error" object on a bare
// `error` frame. Regression test: this used to surface as "unknown (no message
// in error event)", hiding the real upstream failure.
func TestResponsesToChatSSEErrorFrameSurfacesMessage(t *testing.T) {
	in := sse(`{"type":"error","error":{"type":"server_error","code":"server_error","message":"An error occurred while processing your request."},"sequence_number":3}`)
	_, err := readChatCompletionsStream(ResponsesToChatSSE(strings.NewReader(in)), nil)
	if err == nil {
		t.Fatal("expected error from error frame")
	}
	if !strings.Contains(err.Error(), "An error occurred while processing your request.") {
		t.Fatalf("error message not surfaced, got: %v", err)
	}
}
